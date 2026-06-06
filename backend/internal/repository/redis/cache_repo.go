package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheRepository is a Redis-backed generic key/value cache.
type CacheRepository struct {
	client *redis.Client
}

// NewCacheRepository constructs a CacheRepository backed by the given Redis client.
func NewCacheRepository(client *redis.Client) *CacheRepository {
	return &CacheRepository{client: client}
}

// Get returns the cached value for key, or ("", nil) when the key is absent.
func (r *CacheRepository) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return val, err
}

// Set stores value at key with the given TTL.
func (r *CacheRepository) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return r.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes key from the cache.
func (r *CacheRepository) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, key).Err()
}

const passwordResetKeyPrefix = "password_reset"

// passwordResetKey returns the canonical Redis key for a password-reset token.
// Format: "password_reset:{email}"
func passwordResetKey(email string) string {
	return passwordResetKeyPrefix + ":" + email
}

// StorePasswordResetToken persists a password-reset token for the given email
// with the supplied expiry duration.
func (r *CacheRepository) StorePasswordResetToken(ctx context.Context, email, token string, expiry time.Duration) error {
	if err := r.client.Set(ctx, passwordResetKey(email), token, expiry).Err(); err != nil {
		return fmt.Errorf("StorePasswordResetToken: %w", err)
	}
	return nil
}

// GetPasswordResetToken retrieves the stored reset token for the given email.
// Returns ("", nil) when the key is absent or has expired.
func (r *CacheRepository) GetPasswordResetToken(ctx context.Context, email string) (string, error) {
	val, err := r.client.Get(ctx, passwordResetKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("GetPasswordResetToken: %w", err)
	}
	return val, nil
}

// DeletePasswordResetToken removes the password-reset token for the given email.
func (r *CacheRepository) DeletePasswordResetToken(ctx context.Context, email string) error {
	if err := r.client.Del(ctx, passwordResetKey(email)).Err(); err != nil {
		return fmt.Errorf("DeletePasswordResetToken: %w", err)
	}
	return nil
}
