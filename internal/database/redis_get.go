package database

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

func (s *Database) Get(ctx context.Context, path string, version int64) (*Data, error) {
	md5 := getMD5(path)
	metaKey := fmt.Sprintf("meta:%s", md5)

	// * get meta
	data, err := s.RDB.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get meta: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("script not found")
	}

	language := data["language"]
	if language == "" {
		return nil, fmt.Errorf("language not found in meta")
	}

	var assignVersion int64
	// * not assign, use latest
	if version == 0 {
		latestVersion := data["latest"]
		if latestVersion == "" {
			return nil, fmt.Errorf("latest version not found")
		}

		assignVersion, err = strconv.ParseInt(latestVersion, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid version format: %w", err)
		}
	} else {
		assignVersion = version
	}

	scriptKey := fmt.Sprintf("%s:%s:%d", md5, language, assignVersion)
	code, err := s.RDB.Get(ctx, scriptKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("assign version not found")
		}
		return nil, fmt.Errorf("failed to get script: %w", err)
	}

	return &Data{
		Path:      path,
		Code:      code,
		Language:  language,
		Timestamp: assignVersion,
		Hash:      md5,
	}, nil
}
