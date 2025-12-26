package database

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/pardnchiu/go-faas/internal/utils"
	"github.com/redis/go-redis/v9"
)

var (
	DB Database
)

type Database struct {
	RDB *redis.Client
}

type Script struct {
	Path      string
	Code      string
	Language  string
	Timestamp int64
}

func Init() error {
	// * initialize redis RDB with env
	host := utils.GetWithDefault("REDIS_HOST", "localhost")
	port := utils.GetWithDefaultInt("REDIS_PORT", 6379)
	password := utils.GetWithDefault("REDIS_PASSWORD", "")
	dbNum := utils.GetWithDefaultInt("REDIS_DB", 0)
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       dbNum,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return err
	}

	DB = Database{
		RDB: rdb,
	}
	return nil
}

func Close() error {
	return DB.RDB.Close()
}

func (db *Database) Add(ctx context.Context, script Script) (int64, error) {
	hash := md5.Sum([]byte(script.Path))
	hashStr := hex.EncodeToString(hash[:])
	timestamp := time.Now().Unix()

	// * lang not same, can not overwrite
	metaKey := fmt.Sprintf("meta:%s", hashStr)
	codeKey := fmt.Sprintf("code:%s:%d", hashStr, timestamp)
	versionsKey := fmt.Sprintf("%s:version", metaKey)
	// * update meta
	pipe := db.RDB.Pipeline()
	pipe.HSet(ctx, metaKey, map[string]interface{}{
		"path":     script.Path,
		"language": script.Language,
		"latest":   timestamp,
	})

	pipe.Set(ctx, codeKey, script.Code, 0)
	pipe.SAdd(ctx, versionsKey, timestamp)

	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("failed to update meta: %w", err)
	}

	return timestamp, nil
}

func (db *Database) Get(ctx context.Context, path string, version int64) (*Script, error) {
	hash := md5.Sum([]byte(path))
	hashStr := hex.EncodeToString(hash[:])
	metaKey := fmt.Sprintf("meta:%s", hashStr)

	// * get meta
	data, err := db.RDB.HGetAll(ctx, metaKey).Result()
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
	if version == 0 {
		version, _ = db.RDB.HGet(ctx, metaKey, "latest").Int64()
	}

	codeKey := fmt.Sprintf("code:%s:%d", hashStr, version)
	code, err := db.RDB.Get(ctx, codeKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("assign version not found")
		}
		return nil, fmt.Errorf("failed to get script: %w", err)
	}

	return &Script{
		Path:      data["path"],
		Code:      code,
		Language:  data["language"],
		Timestamp: version,
	}, nil
}
