package toplist

import (
	"context"
	"fmt"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

const (
	// ToplistConfigCacheTTL is the TTL for cached toplist configurations (1 hour)
	ToplistConfigCacheTTL = 1 * time.Hour
)

// ToplistService manages toplist operations including ranking computation and updates
type ToplistService struct {
	store      ToplistStore
	redisClient storage.RedisClient
	updater    ToplistUpdater
}

// NewToplistService creates a new toplist service
func NewToplistService(store ToplistStore, redisClient storage.RedisClient, updater ToplistUpdater) *ToplistService {
	return &ToplistService{
		store:      store,
		redisClient: redisClient,
		updater:    updater,
	}
}

// GetToplistRankings retrieves rankings for a toplist with optional filters
func (s *ToplistService) GetToplistRankings(ctx context.Context, toplistID string, limit int, offset int) ([]models.ToplistRanking, error) {
	// Load toplist configuration
	config, err := s.getToplistConfig(ctx, toplistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get toplist config: %w", err)
	}

	// Determine Redis key
	var redisKey string
	if config.IsSystemToplist() {
		redisKey = models.GetSystemToplistRedisKey(config.Metric, config.TimeWindow)
	} else {
		redisKey = models.GetUserToplistRedisKey(config.UserID, toplistID)
	}

	// Get rankings from Redis ZSET
	start := int64(offset)
	stop := int64(offset + limit - 1)
	if config.SortOrder == models.SortOrderAsc {
		// For ascending, we need to get from the beginning and reverse
		// For now, we'll use ZREVRANGE and reverse if needed
		// In production, we might want to use ZRANGE for ascending
		members, err := s.redisClient.ZRevRange(ctx, redisKey, start, stop)
		if err != nil {
			return nil, fmt.Errorf("failed to get rankings from Redis: %w", err)
		}

		rankings := make([]models.ToplistRanking, len(members))
		for i, member := range members {
			rankings[i] = models.ToplistRanking{
				Symbol:   member.Member,
				Rank:     offset + i + 1,
				Value:    member.Score,
				Metadata: make(map[string]interface{}),
			}
		}

		// Reverse if ascending
		if config.SortOrder == models.SortOrderAsc {
			for i, j := 0, len(rankings)-1; i < j; i, j = i+1, j-1 {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}

		// Apply filters if specified
		if config.Filters != nil {
			rankings = s.applyFilters(rankings, config.Filters)
		}

		return rankings, nil
	} else {
		// Descending order (default for ZREVRANGE)
		members, err := s.redisClient.ZRevRange(ctx, redisKey, start, stop)
		if err != nil {
			return nil, fmt.Errorf("failed to get rankings from Redis: %w", err)
		}

		rankings := make([]models.ToplistRanking, len(members))
		for i, member := range members {
			rankings[i] = models.ToplistRanking{
				Symbol:   member.Member,
				Rank:     offset + i + 1,
				Value:    member.Score,
				Metadata: make(map[string]interface{}),
			}
		}

		// Apply filters if specified
		if config.Filters != nil {
			rankings = s.applyFilters(rankings, config.Filters)
		}

		return rankings, nil
	}
}

// getToplistConfig retrieves a toplist config, checking cache first
func (s *ToplistService) getToplistConfig(ctx context.Context, toplistID string) (*models.ToplistConfig, error) {
	// Try cache first
	cacheKey := models.GetToplistConfigRedisKey(toplistID)
	var config models.ToplistConfig
	err := s.redisClient.GetJSON(ctx, cacheKey, &config)
	if err == nil && config.ID != "" {
		return &config, nil
	}

		// Cache miss, load from database
		configPtr, err := s.store.GetToplistConfig(ctx, toplistID)
		if err != nil {
			return nil, err
		}

		// Cache the config
		if err := s.redisClient.Set(ctx, cacheKey, configPtr, ToplistConfigCacheTTL); err != nil {
			logger.Warn("Failed to cache toplist config",
				logger.ErrorField(err),
				logger.String("toplist_id", toplistID),
			)
			// Don't fail if caching fails
		}

		return configPtr, nil
}

// applyFilters applies filters to rankings
func (s *ToplistService) applyFilters(rankings []models.ToplistRanking, filters *models.ToplistFilter) []models.ToplistRanking {
	if filters == nil {
		return rankings
	}

	filtered := make([]models.ToplistRanking, 0, len(rankings))
	for _, ranking := range rankings {
		// Apply filters based on metadata
		// Note: In a real implementation, we'd need to fetch additional data
		// (price, volume, etc.) from Redis or database to apply filters
		// For now, we'll just return all rankings
		// TODO: Implement proper filtering when metadata is available
		filtered = append(filtered, ranking)
	}

	return filtered
}

// ProcessToplistUpdate processes a toplist update notification
func (s *ToplistService) ProcessToplistUpdate(ctx context.Context, toplistID string, toplistType string) error {
	// For user toplists, we need to recompute rankings based on the config
	if toplistType == "user" {
		_, err := s.getToplistConfig(ctx, toplistID)
		if err != nil {
			return fmt.Errorf("failed to get toplist config: %w", err)
		}

		// Publish update notification
		if err := s.updater.PublishUpdate(ctx, toplistID, toplistType); err != nil {
			logger.Warn("Failed to publish toplist update",
				logger.ErrorField(err),
				logger.String("toplist_id", toplistID),
			)
		}
	}

	return nil
}

// GetEnabledToplists returns all enabled toplists
func (s *ToplistService) GetEnabledToplists(ctx context.Context) ([]*models.ToplistConfig, error) {
	return s.store.GetEnabledToplists(ctx)
}

// Close closes the service
func (s *ToplistService) Close() error {
	return s.store.Close()
}

