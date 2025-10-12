package database

import (
	"context"
	"fmt"
	"time"
)

func (s *Database) Add(ctx context.Context, path, code, language string) (int64, error) {
	md5 := getMD5(path)
	timestamp := time.Now().Unix()

	// * lang not same, can not overwrite
	metaKey := fmt.Sprintf("meta:%s", md5)
	exist, err := s.RDB.HGet(ctx, metaKey, "language").Result()
	if err == nil && exist != "" && exist != language {
		return 0, fmt.Errorf("path already used: %s", exist)
	}

	// * save script code
	scriptKey := fmt.Sprintf("%s:%s:%d", md5, language, timestamp)
	if err := s.RDB.Set(ctx, scriptKey, code, 0).Err(); err != nil {
		return 0, fmt.Errorf("failed to save script: %w", err)
	}

	// * update meta
	pipe := s.RDB.Pipeline()
	pipe.HSet(ctx, metaKey, "path", path)
	pipe.HSet(ctx, metaKey, "hash", md5)
	pipe.HSet(ctx, metaKey, "language", language)
	pipe.HSet(ctx, metaKey, "latest", timestamp)

	// TODO: record version for future use
	pipe.SAdd(ctx, fmt.Sprintf("%s:version", metaKey), timestamp)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to update meta: %w", err)
	}

	return timestamp, nil
}
