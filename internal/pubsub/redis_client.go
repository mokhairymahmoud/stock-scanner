package pubsub

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/mohamedkhairy/stock-scanner/internal/config"
	"github.com/mohamedkhairy/stock-scanner/internal/storage"
	"github.com/mohamedkhairy/stock-scanner/pkg/logger"
)

// RedisClientImpl implements the storage.RedisClient interface
type RedisClientImpl struct {
	client *redis.Client
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg config.RedisConfig) (storage.RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Info("Connected to Redis",
		logger.String("host", cfg.Host),
		logger.Int("port", cfg.Port),
	)

	return &RedisClientImpl{client: rdb}, nil
}

// PublishToStream publishes a message to a Redis stream
func (r *RedisClientImpl) PublishToStream(ctx context.Context, stream string, key string, value interface{}) error {
	// Serialize value to JSON
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Publish to stream with key as field name
	err = r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			key: string(jsonData),
		},
	}).Err()

	if err != nil {
		return fmt.Errorf("failed to publish to stream %s: %w", stream, err)
	}

	return nil
}

// PublishBatchToStream publishes multiple messages to a Redis stream using a pipeline
func (r *RedisClientImpl) PublishBatchToStream(ctx context.Context, stream string, messages []map[string]interface{}) error {
	if len(messages) == 0 {
		return nil
	}

	// Use pipeline for batch operations
	pipe := r.client.Pipeline()

	for _, msg := range messages {
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: stream,
			Values: msg,
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish batch to stream %s: %w", stream, err)
	}

	return nil
}

// ConsumeFromStream consumes messages from a Redis stream
func (r *RedisClientImpl) ConsumeFromStream(ctx context.Context, stream string, group string, consumer string) (<-chan storage.StreamMessage, error) {
	messageChan := make(chan storage.StreamMessage, 100)

	// Create consumer group if it doesn't exist (with retry)
	// XGroupCreateMkStream creates the stream if it doesn't exist (MKSTREAM)
	var groupCreated bool
	for i := 0; i < 3; i++ {
		err := r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
		if err == nil {
			groupCreated = true
			logger.Debug("Created consumer group",
				logger.String("stream", stream),
				logger.String("group", group),
			)
			break
		}
		// BUSYGROUP means group already exists - that's OK
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			groupCreated = true
			logger.Debug("Consumer group already exists",
				logger.String("stream", stream),
				logger.String("group", group),
			)
			break
		}
		// For other errors, retry after a short delay
		logger.Warn("Failed to create consumer group, retrying",
			logger.ErrorField(err),
			logger.String("stream", stream),
			logger.String("group", group),
			logger.Int("attempt", i+1),
		)
		time.Sleep(time.Second * time.Duration(i+1))
	}

	if !groupCreated {
		logger.Error("Failed to create consumer group after retries",
			logger.String("stream", stream),
			logger.String("group", group),
		)
		// Continue anyway - will retry in the read loop
	}

	go func() {
		defer close(messageChan)

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Read from stream
			streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    group,
				Consumer: consumer,
				Streams:  []string{stream, ">"},
				Count:    10,
				Block:    time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue
				}
				
				// Handle NOGROUP error - try to recreate the group
				errStr := err.Error()
				if strings.Contains(errStr, "NOGROUP") {
					logger.Warn("Consumer group not found, attempting to create",
						logger.String("stream", stream),
						logger.String("group", group),
					)
					// Try to create the group again
					createErr := r.client.XGroupCreateMkStream(ctx, stream, group, "0").Err()
					if createErr != nil && createErr.Error() != "BUSYGROUP Consumer Group name already exists" {
						logger.Error("Failed to recreate consumer group",
							logger.ErrorField(createErr),
							logger.String("stream", stream),
							logger.String("group", group),
						)
					}
					// Wait a bit before retrying
					time.Sleep(2 * time.Second)
					continue
				}
				
				logger.Error("Error reading from stream",
					logger.ErrorField(err),
					logger.String("stream", stream),
				)
				time.Sleep(time.Second)
				continue
			}

			for _, stream := range streams {
				for _, message := range stream.Messages {
					msg := storage.StreamMessage{
						ID:     message.ID,
						Stream: stream.Stream,
						Values: message.Values,
					}
					select {
					case messageChan <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	return messageChan, nil
}

// AcknowledgeMessage acknowledges a message in a Redis stream
func (r *RedisClientImpl) AcknowledgeMessage(ctx context.Context, stream string, group string, id string) error {
	return r.client.XAck(ctx, stream, group, id).Err()
}

// Set sets a key-value pair with TTL
func (r *RedisClientImpl) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return r.client.Set(ctx, key, jsonData, ttl).Err()
}

// Get gets a value by key
func (r *RedisClientImpl) Get(ctx context.Context, key string) (string, error) {
	result, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}
	return result, err
}

// GetJSON gets a JSON value and unmarshals it
func (r *RedisClientImpl) GetJSON(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete deletes a key
func (r *RedisClientImpl) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

// Exists checks if a key exists
func (r *RedisClientImpl) Exists(ctx context.Context, key string) (bool, error) {
	count, err := r.client.Exists(ctx, key).Result()
	return count > 0, err
}

// SetAdd adds members to a set
func (r *RedisClientImpl) SetAdd(ctx context.Context, key string, members ...string) error {
	return r.client.SAdd(ctx, key, members).Err()
}

// SetMembers gets all members of a set
func (r *RedisClientImpl) SetMembers(ctx context.Context, key string) ([]string, error) {
	return r.client.SMembers(ctx, key).Result()
}

// SetRemove removes members from a set
func (r *RedisClientImpl) SetRemove(ctx context.Context, key string, members ...string) error {
	return r.client.SRem(ctx, key, members).Err()
}

// Publish publishes a message to a pub/sub channel
func (r *RedisClientImpl) Publish(ctx context.Context, channel string, message interface{}) error {
	jsonData, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return r.client.Publish(ctx, channel, jsonData).Err()
}

// Subscribe subscribes to pub/sub channels
func (r *RedisClientImpl) Subscribe(ctx context.Context, channels ...string) (<-chan storage.PubSubMessage, error) {
	pubsub := r.client.Subscribe(ctx, channels...)
	messageChan := make(chan storage.PubSubMessage, 100)

	go func() {
		defer close(messageChan)
		ch := pubsub.Channel()

		for {
			select {
			case <-ctx.Done():
				pubsub.Close()
				return
			case msg := <-ch:
				if msg == nil {
					return
				}
				psMsg := storage.PubSubMessage{
					Channel: msg.Channel,
					Message: msg.Payload,
				}
				select {
				case messageChan <- psMsg:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return messageChan, nil
}

// Close closes the Redis connection
func (r *RedisClientImpl) Close() error {
	return r.client.Close()
}

