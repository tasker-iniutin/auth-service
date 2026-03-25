package redis

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	d "github.com/tasker-iniutin/auth-service/internal/domain"

	"github.com/redis/go-redis/v9"
)

type redisRepoImpl struct {
	rdb       *redis.Client
	keyPrefix string
}

func NewRedisRepo(rdp *redis.Client) *redisRepoImpl {
	return &redisRepoImpl{
		rdb:       rdp,
		keyPrefix: "rt",
	}
}

func (r *redisRepoImpl) fmtKey(id d.SessionID) string {
	return fmt.Sprintf("%s:%s", r.keyPrefix, id)
}

func (r *redisRepoImpl) hashKey(tokenHash []byte) string {
	return fmt.Sprintf("%s_h:%s", r.keyPrefix, hex.EncodeToString(tokenHash))
}

func (r *redisRepoImpl) CreateRefresh(ctx context.Context, s d.RefreshSession) error {
	if s.ID == "" || s.UserID == 0 || len(s.TokenHash) == 0 {
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
	pipe.Set(ctx, r.hashKey(s.TokenHash), string(s.ID), ttl)
	pipe.Set(ctx, r.fmtKey(s.ID), b, ttl)
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

	sid, err := r.rdb.Get(ctx, r.hashKey(tokenHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return d.RefreshSession{}, d.ErrNotFound
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

	if len(s.TokenHash) == 0 || !bytes.Equal(s.TokenHash, tokenHash) {
		return d.RefreshSession{}, d.ErrNotFound
	}

	return s, nil
}

func (r *redisRepoImpl) RevokeRefresh(ctx context.Context, tokenHash []byte) error {
	if len(tokenHash) == 0 {
		return errors.New("empty tokenHash")
	}

	sid, err := r.rdb.Get(ctx, r.hashKey(tokenHash)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil
		}
		return fmt.Errorf("redis get session id by hash: %w", err)
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, r.hashKey(tokenHash))
	pipe.Del(ctx, r.fmtKey(d.SessionID(sid)))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("redis revoke: %w", err)
	}

	return nil
}
