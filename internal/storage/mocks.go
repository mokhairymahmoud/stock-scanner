package storage

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/mohamedkhairy/stock-scanner/internal/models"
)

// MockBarStorage is a mock implementation of BarStorage for testing
type MockBarStorage struct {
	Bars      []*models.Bar1m
	WriteErr  error
	GetErr    error
	LatestErr error
}

func (m *MockBarStorage) WriteBars(ctx context.Context, bars []*models.Bar1m) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Bars = append(m.Bars, bars...)
	return nil
}

func (m *MockBarStorage) GetBars(ctx context.Context, symbol string, start, end time.Time) ([]*models.Bar1m, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	var result []*models.Bar1m
	for _, bar := range m.Bars {
		if bar.Symbol == symbol && !bar.Timestamp.Before(start) && !bar.Timestamp.After(end) {
			result = append(result, bar)
		}
	}
	return result, nil
}

func (m *MockBarStorage) GetLatestBars(ctx context.Context, symbol string, limit int) ([]*models.Bar1m, error) {
	if m.LatestErr != nil {
		return nil, m.LatestErr
	}
	var result []*models.Bar1m
	for i := len(m.Bars) - 1; i >= 0 && len(result) < limit; i-- {
		if m.Bars[i].Symbol == symbol {
			result = append(result, m.Bars[i])
		}
	}
	// Reverse to get chronological order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

func (m *MockBarStorage) Close() error {
	return nil
}

// MockAlertStorage is a mock implementation of AlertStorage for testing
type MockAlertStorage struct {
	Alerts   []*models.Alert
	WriteErr error
	GetErr   error
}

func (m *MockAlertStorage) WriteAlert(ctx context.Context, alert *models.Alert) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Alerts = append(m.Alerts, alert)
	return nil
}

func (m *MockAlertStorage) WriteAlerts(ctx context.Context, alerts []*models.Alert) error {
	if m.WriteErr != nil {
		return m.WriteErr
	}
	m.Alerts = append(m.Alerts, alerts...)
	return nil
}

func (m *MockAlertStorage) GetAlerts(ctx context.Context, filter AlertFilter) ([]*models.Alert, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	var result []*models.Alert
	for _, alert := range m.Alerts {
		if filter.Symbol != "" && alert.Symbol != filter.Symbol {
			continue
		}
		if filter.RuleID != "" && alert.RuleID != filter.RuleID {
			continue
		}
		if !filter.StartTime.IsZero() && alert.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && alert.Timestamp.After(filter.EndTime) {
			continue
		}
		result = append(result, alert)
	}
	// Apply limit and offset
	start := filter.Offset
	if start > len(result) {
		start = len(result)
	}
	end := start + filter.Limit
	if end > len(result) {
		end = len(result)
	}
	if filter.Limit > 0 {
		return result[start:end], nil
	}
	return result[start:], nil
}

func (m *MockAlertStorage) GetAlert(ctx context.Context, alertID string) (*models.Alert, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	for _, alert := range m.Alerts {
		if alert.ID == alertID {
			return alert, nil
		}
	}
	return nil, nil
}

func (m *MockAlertStorage) Close() error {
	return nil
}

// MockRedisClient is a mock implementation of RedisClient for testing
type MockRedisClient struct {
	Data          map[string]string
	Sets          map[string]map[string]bool // Map of set keys to their members
	ZSets         map[string]map[string]float64 // Map of ZSET keys to member->score mappings
	StreamData    []StreamMessage
	PubSubData    []PubSubMessage
	PublishErr    error
	GetErr        error
	SetErr        error
	SubscribeErr  error
	ConsumeErr    error
	mu            sync.RWMutex
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		Data:  make(map[string]string),
		Sets:  make(map[string]map[string]bool),
		ZSets: make(map[string]map[string]float64),
	}
}

func (m *MockRedisClient) PublishToStream(ctx context.Context, stream string, key string, value interface{}) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	// Mock implementation
	return nil
}

func (m *MockRedisClient) PublishBatchToStream(ctx context.Context, stream string, messages []map[string]interface{}) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	// Store messages in StreamData for testing
	for _, msg := range messages {
		// Convert map to StreamMessage format
		streamMsg := StreamMessage{
			ID:     "", // Mock doesn't generate IDs
			Stream: stream,
			Values: msg,
		}
		m.StreamData = append(m.StreamData, streamMsg)
	}
	return nil
}

func (m *MockRedisClient) ConsumeFromStream(ctx context.Context, stream string, group string, consumer string) (<-chan StreamMessage, error) {
	if m.ConsumeErr != nil {
		return nil, m.ConsumeErr
	}
	ch := make(chan StreamMessage, len(m.StreamData))
	for _, msg := range m.StreamData {
		ch <- msg
	}
	close(ch)
	return ch, nil
}

func (m *MockRedisClient) AcknowledgeMessage(ctx context.Context, stream string, group string, id string) error {
	return nil
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.SetErr != nil {
		return m.SetErr
	}
	// Marshal to JSON like the real implementation
	jsonData, err := json.Marshal(value)
	if err != nil {
		return err
	}
	m.Data[key] = string(jsonData)
	return nil
}

func (m *MockRedisClient) Get(ctx context.Context, key string) (string, error) {
	if m.GetErr != nil {
		return "", m.GetErr
	}
	return m.Data[key], nil
}

func (m *MockRedisClient) GetJSON(ctx context.Context, key string, dest interface{}) error {
	if m.GetErr != nil {
		return m.GetErr
	}
	value, exists := m.Data[key]
	if !exists {
		return nil // Return nil if key doesn't exist (like real implementation)
	}
	return json.Unmarshal([]byte(value), dest)
}

func (m *MockRedisClient) Delete(ctx context.Context, key string) error {
	delete(m.Data, key)
	return nil
}

func (m *MockRedisClient) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := m.Data[key]
	return exists, nil
}

func (m *MockRedisClient) SetAdd(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Sets[key] == nil {
		m.Sets[key] = make(map[string]bool)
	}
	for _, member := range members {
		m.Sets[key][member] = true
	}
	return nil
}

func (m *MockRedisClient) SetMembers(ctx context.Context, key string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	set, exists := m.Sets[key]
	if !exists {
		return []string{}, nil
	}
	members := make([]string, 0, len(set))
	for member := range set {
		members = append(members, member)
	}
	return members, nil
}

func (m *MockRedisClient) SetRemove(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	set, exists := m.Sets[key]
	if !exists {
		return nil
	}
	for _, member := range members {
		delete(set, member)
	}
	return nil
}

func (m *MockRedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	if m.PublishErr != nil {
		return m.PublishErr
	}
	return nil
}

func (m *MockRedisClient) Subscribe(ctx context.Context, channels ...string) (<-chan PubSubMessage, error) {
	if m.SubscribeErr != nil {
		return nil, m.SubscribeErr
	}
	ch := make(chan PubSubMessage, len(m.PubSubData))
	for _, msg := range m.PubSubData {
		ch <- msg
	}
	close(ch)
	return ch, nil
}

func (m *MockRedisClient) ZAdd(ctx context.Context, key string, score float64, member string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ZSets[key] == nil {
		m.ZSets[key] = make(map[string]float64)
	}
	m.ZSets[key][member] = score
	return nil
}

func (m *MockRedisClient) ZAddBatch(ctx context.Context, key string, members map[string]float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ZSets[key] == nil {
		m.ZSets[key] = make(map[string]float64)
	}
	for member, score := range members {
		m.ZSets[key][member] = score
	}
	return nil
}

func (m *MockRedisClient) ZRevRange(ctx context.Context, key string, start, stop int64) ([]ZSetMember, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	zset, exists := m.ZSets[key]
	if !exists {
		return []ZSetMember{}, nil
	}

	// Convert map to slice and sort by score (descending)
	type memberScore struct {
		member string
		score  float64
	}
	members := make([]memberScore, 0, len(zset))
	for member, score := range zset {
		members = append(members, memberScore{member: member, score: score})
	}

	// Simple sort by score (descending)
	for i := 0; i < len(members); i++ {
		for j := i + 1; j < len(members); j++ {
			if members[i].score < members[j].score {
				members[i], members[j] = members[j], members[i]
			}
		}
	}

	// Apply start/stop range
	if start < 0 {
		start = 0
	}
	if stop < 0 || stop >= int64(len(members)) {
		stop = int64(len(members)) - 1
	}
	if start > stop {
		return []ZSetMember{}, nil
	}

	result := make([]ZSetMember, 0, stop-start+1)
	for i := start; i <= stop && i < int64(len(members)); i++ {
		result = append(result, ZSetMember{
			Member: members[i].member,
			Score:  members[i].score,
		})
	}

	return result, nil
}

func (m *MockRedisClient) ZRem(ctx context.Context, key string, members ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	zset, exists := m.ZSets[key]
	if !exists {
		return nil
	}
	for _, member := range members {
		delete(zset, member)
	}
	return nil
}

func (m *MockRedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	zset, exists := m.ZSets[key]
	if !exists {
		return 0, nil
	}
	return int64(len(zset)), nil
}

func (m *MockRedisClient) ZScore(ctx context.Context, key string, member string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	zset, exists := m.ZSets[key]
	if !exists {
		return 0, nil
	}
	score, exists := zset[member]
	if !exists {
		return 0, nil
	}
	return score, nil
}

func (m *MockRedisClient) Close() error {
	return nil
}

