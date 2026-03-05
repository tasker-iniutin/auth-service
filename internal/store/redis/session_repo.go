package redis

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	d "todo/auth-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type redisRepoImpl struct {
	rdb       *redis.Client
	keyPrefix string // "rt" or "sess"
}

func NewRedisRepo(rdp *redis.Client) *redisRepoImpl {
	return &redisRepoImpl{
		rdb:       rdp,
		keyPrefix: "rt",
	}
}

func (r *redisRepoImpl) fmtKey(id d.SessionID) string {
	return fmt.Sprintf("%s:%d", r.keyPrefix, id)
}
func (r *redisRepoImpl) hashKey(tokenHash []byte) string {
	return fmt.Sprintf("%s_h:%s", r.keyPrefix, hex.EncodeToString(tokenHash))
}
func (r *redisRepoImpl) userIndexKey(id d.UserID) string {
	return fmt.Sprintf("%s_u:%d", r.keyPrefix, id) // set с sessionID
}

func (r *redisRepoImpl) CreateRefresh(ctx context.Context, s d.RefreshSession) error {
	if s.ID == 0 || s.UserID == 0 || s.TokenHash == nil {
		return errors.New("empty required fields")
	}
	ttl := time.Until(s.ExpiresAt)
	if ttl <= 0 {
		_ = r.RevokeRefresh(ctx, s.TokenHash)
		return nil
	}

	b, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, r.fmtKey(s.ID), b, ttl)
	pipe.SAdd(ctx, r.userIndexKey(s.UserID), s.ID)
	pipe.Expire(ctx, r.userIndexKey(s.UserID), ttl)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis upsert: %w", err)
	}
	return nil
}
func (r *redisRepoImpl) GetRefresh(ctx context.Context, tokenHash []byte) (d.RefreshSession, error) {
	if len(tokenHash) == 0 {
		return d.RefreshSession{}, errors.New("empty tokenHash")
	}

	// 1) tokenHash -> sessionID
	sid, err := r.rdb.Get(ctx, r.hashKey(tokenHash)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return d.RefreshSession{}, d.ErrNotFound // или как у тебя принято
		}
		return d.RefreshSession{}, fmt.Errorf("redis get session id by hash: %w", err)
	}

	val, err := r.rdb.Get(ctx, r.fmtKey(d.SessionID(sid))).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			_ = r.rdb.Del(ctx, r.hashKey(tokenHash)).Err()
			return d.RefreshSession{}, d.ErrNotFound
		}
		return d.RefreshSession{}, fmt.Errorf("redis get session: %w", err)
	}

	var s d.RefreshSession
	if err := json.Unmarshal(val, &s); err != nil {
		return d.RefreshSession{}, fmt.Errorf("unmarshal session: %w", err)
	}

	if len(s.TokenHash) == 0 || hex.EncodeToString(s.TokenHash) != hex.EncodeToString(tokenHash) {
		return d.RefreshSession{}, d.ErrNotFound
	}

	return s, nil
}
func (r *redisRepoImpl) RevokeRefresh(ctx context.Context, tokenHash []byte) error {
	if len(tokenHash) == 0 {
		return errors.New("empty tokenHash")
	}

	sid, err := r.rdb.Get(ctx, r.hashKey(tokenHash)).Int64()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("redis get session id by hash: %w", err)
	}

	var s d.RefreshSession
	if raw, err := r.rdb.Get(ctx, r.fmtKey(d.SessionID(sid))).Bytes(); err == nil {
		_ = json.Unmarshal(raw, &s)
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, r.hashKey(tokenHash))
	pipe.Del(ctx, r.fmtKey(d.SessionID(sid)))
	if s.UserID != 0 {
		pipe.SRem(ctx, r.userIndexKey(s.UserID), sid)
	}
	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis revoke: %w", err)
	}
	return nil
}
