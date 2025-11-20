package toplist

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// ToplistService manages toplist operations including ranking computation
type ToplistService struct {
	store       ToplistStore
	redisClient storage.RedisClient
	updater     ToplistUpdater
}

// NewToplistService creates a new toplist service
func NewToplistService(store ToplistStore, redisClient storage.RedisClient, updater ToplistUpdater) *ToplistService {
	return &ToplistService{
		store:       store,
		redisClient: redisClient,
		updater:     updater,
	}
}

// GetToplistRankings retrieves rankings for a toplist with optional filtering
func (s *ToplistService) GetToplistRankings(ctx context.Context, toplistID string, limit, offset int, filters *models.ToplistFilter) ([]models.ToplistRanking, error) {
	// Get toplist configuration
	config, err := s.store.GetToplistConfig(ctx, toplistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get toplist config: %w", err)
	}

	return s.GetRankingsByConfig(ctx, config, limit, offset, filters)
}

// GetRankingsByConfig retrieves rankings using a config directly (for system toplists)
func (s *ToplistService) GetRankingsByConfig(ctx context.Context, config *models.ToplistConfig, limit, offset int, filters *models.ToplistFilter) ([]models.ToplistRanking, error) {
	// Determine Redis key
	var redisKey string
	if config.IsSystemToplist() {
		redisKey = models.GetSystemToplistRedisKey(config.Metric, config.TimeWindow)
	} else {
		redisKey = models.GetUserToplistRedisKey(config.UserID, config.ID)
	}

	// Get rankings from Redis ZSET
	// For descending order, use ZRevRange (highest to lowest)
	// For ascending order, we'd need ZRange, but for now we'll use ZRevRange and reverse if needed
	start := int64(offset)
	stop := int64(offset + limit - 1)
	if stop < 0 {
		stop = 0
	}

	members, err := s.redisClient.ZRevRange(ctx, redisKey, start, stop)
	if err != nil {
		return nil, fmt.Errorf("failed to get rankings from Redis: %w", err)
	}

	// Convert to ToplistRanking format
	rankings := make([]models.ToplistRanking, 0, len(members))
	for i, member := range members {
		ranking := models.ToplistRanking{
			Symbol:   member.Member,
			Rank:     offset + i + 1,
			Value:    member.Score,
			Metadata: make(map[string]interface{}),
		}

		// Apply filters if provided
		if filters != nil {
			// Note: For MVP, we'll do basic filtering here
			// In production, you might want to pre-filter in Redis or use a more sophisticated approach
			if filters.MinVolume != nil {
				// Would need to fetch volume from symbol data
				// For now, we'll skip this filter or implement it later
			}
			if filters.PriceMin != nil || filters.PriceMax != nil {
				// Would need to fetch price from symbol data
				// For now, we'll skip this filter or implement it later
			}
		}

		rankings = append(rankings, ranking)
	}

	// If sort order is ascending, reverse the results
	if config.SortOrder == models.SortOrderAsc {
		for i, j := 0, len(rankings)-1; i < j; i, j = i+1, j-1 {
			rankings[i], rankings[j] = rankings[j], rankings[i]
		}
	}

	return rankings, nil
}

// GetToplistCount returns the total number of symbols in a toplist
func (s *ToplistService) GetToplistCount(ctx context.Context, toplistID string) (int64, error) {
	// Get toplist configuration
	config, err := s.store.GetToplistConfig(ctx, toplistID)
	if err != nil {
		return 0, fmt.Errorf("failed to get toplist config: %w", err)
	}

	return s.GetCountByConfig(ctx, config)
}

// GetCountByConfig returns the count using a config directly (for system toplists)
func (s *ToplistService) GetCountByConfig(ctx context.Context, config *models.ToplistConfig) (int64, error) {
	// Determine Redis key
	var redisKey string
	if config.IsSystemToplist() {
		redisKey = models.GetSystemToplistRedisKey(config.Metric, config.TimeWindow)
	} else {
		redisKey = models.GetUserToplistRedisKey(config.UserID, config.ID)
	}

	// Get count from Redis
	count, err := s.redisClient.ZCard(ctx, redisKey)
	if err != nil {
		return 0, fmt.Errorf("failed to get toplist count: %w", err)
	}

	return count, nil
}

// CacheToplistConfig caches a toplist configuration in Redis
func (s *ToplistService) CacheToplistConfig(ctx context.Context, config *models.ToplistConfig) error {
	key := models.GetToplistConfigRedisKey(config.ID)
	ttl := 1 * time.Hour

	err := s.redisClient.Set(ctx, key, config, ttl)
	if err != nil {
		logger.Warn("Failed to cache toplist config",
			logger.ErrorField(err),
			logger.String("toplist_id", config.ID),
		)
		return err
	}

	return nil
}

// GetCachedToplistConfig retrieves a cached toplist configuration from Redis
func (s *ToplistService) GetCachedToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error) {
	key := models.GetToplistConfigRedisKey(toplistID)
	var config models.ToplistConfig

	err := s.redisClient.GetJSON(ctx, key, &config)
	if err != nil {
		return nil, err
	}

	// Check if config was found
	if config.ID == "" {
		return nil, nil
	}

	return &config, nil
}

// RefreshToplistCache refreshes the cache for a toplist configuration
func (s *ToplistService) RefreshToplistCache(ctx context.Context, toplistID string) error {
	config, err := s.store.GetToplistConfig(ctx, toplistID)
	if err != nil {
		return err
	}

	return s.CacheToplistConfig(ctx, config)
}

// ProcessToplistUpdate processes a toplist update notification and republishes if needed
func (s *ToplistService) ProcessToplistUpdate(ctx context.Context, toplistID string, toplistType string) error {
	// For user toplists, we might need to recompute rankings
	// For now, we'll just republish the update
	if toplistType == "user" {
		// Verify the toplist exists and is enabled
		config, err := s.store.GetToplistConfig(ctx, toplistID)
		if err != nil {
			logger.Debug("Toplist not found, skipping update",
				logger.String("toplist_id", toplistID),
			)
			return nil
		}

		if !config.Enabled {
			return nil
		}

		// Republish update notification
		return s.updater.PublishUpdate(ctx, toplistID, toplistType)
	}

	return nil
}

// Close closes the service
func (s *ToplistService) Close() error {
	if s.store != nil {
		return s.store.Close()
	}
	return nil
}
