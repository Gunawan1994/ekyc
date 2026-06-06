package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monarchintiteknologi/ekyc-platform/internal/domain"
	"github.com/redis/go-redis/v9"
)

const refreshTokenKeyPrefix = "refresh"

// TokenRepository provides Redis-backed storage for refresh tokens and general
// key-value caching.
type TokenRepository struct {
	client *redis.Client
}

// NewTokenRepository returns a TokenRepository backed by the given Redis client.
func NewTokenRepository(client *redis.Client) *TokenRepository {
	return &TokenRepository{client: client}
}

// refreshKey returns the canonical Redis key for a refresh token.
// Format: "refresh:{userID}:{tokenID}"
func refreshKey(userID, tokenID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:%s", refreshTokenKeyPrefix, userID, tokenID)
}

// userRefreshPattern returns the glob pattern that matches all refresh keys for
// a given user.  Used by SCAN when revoking all sessions.
func userRefreshPattern(userID uuid.UUID) string {
	return fmt.Sprintf("%s:%s:*", refreshTokenKeyPrefix, userID)
}

// StoreRefreshToken persists the token hash under "refresh:{userID}:{tokenID}"
// with the given expiry duration.
func (r *TokenRepository) StoreRefreshToken(
	ctx context.Context,
	userID, tokenID uuid.UUID,
	tokenHash string,
	expiry time.Duration,
) error {
	key := refreshKey(userID, tokenID)
	if err := r.client.Set(ctx, key, tokenHash, expiry).Err(); err != nil {
		return fmt.Errorf("StoreRefreshToken: %w", err)
	}
	return nil
}

// ValidateRefreshToken returns nil when the token exists and the stored hash
// matches tokenHash.  It returns domain.ErrTokenRevoked when the key is
// missing (expired or deleted) and domain.ErrTokenInvalid when the hash does
// not match.
func (r *TokenRepository) ValidateRefreshToken(
	ctx context.Context,
	userID, tokenID uuid.UUID,
	tokenHash string,
) error {
	key := refreshKey(userID, tokenID)

	stored, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.ErrTokenRevoked
		}
		return fmt.Errorf("ValidateRefreshToken: %w", err)
	}

	if stored != tokenHash {
		return domain.ErrTokenInvalid
	}
	return nil
}

// RevokeRefreshToken deletes a single refresh token key.
func (r *TokenRepository) RevokeRefreshToken(
	ctx context.Context,
	userID, tokenID uuid.UUID,
) error {
	key := refreshKey(userID, tokenID)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("RevokeRefreshToken: %w", err)
	}
	return nil
}

// RevokeAllUserTokens removes every refresh token belonging to userID.  It
// uses an iterative SCAN to avoid blocking the server with a KEYS call on
// large datasets.
func (r *TokenRepository) RevokeAllUserTokens(
	ctx context.Context,
	userID uuid.UUID,
) error {
	pattern := userRefreshPattern(userID)
	var cursor uint64
	const scanCount int64 = 100

	for {
		keys, next, err := r.client.Scan(ctx, cursor, pattern, scanCount).Result()
		if err != nil {
			return fmt.Errorf("RevokeAllUserTokens scan: %w", err)
		}

		if len(keys) > 0 {
			if err := r.client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("RevokeAllUserTokens del: %w", err)
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}

// SetCache JSON-serializes value and stores it under key with the given TTL.
// Pass ttl = 0 for no expiry.
func (r *TokenRepository) SetCache(
	ctx context.Context,
	key string,
	value interface{},
	ttl time.Duration,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("SetCache marshal: %w", err)
	}
	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("SetCache set: %w", err)
	}
	return nil
}

// GetCache retrieves the value stored under key and JSON-deserializes it into
// dest.  Returns domain.ErrNotFound when the key does not exist or has expired.
func (r *TokenRepository) GetCache(
	ctx context.Context,
	key string,
	dest interface{},
) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("GetCache get: %w", err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("GetCache unmarshal: %w", err)
	}
	return nil
}

// DeleteCache removes one or more cache keys.  Missing keys are silently
// ignored by Redis and do not produce an error.
func (r *TokenRepository) DeleteCache(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if err := r.client.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("DeleteCache: %w", err)
	}
	return nil
}
