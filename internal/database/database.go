package database

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	DB Database
)

type Config struct {
	Redis *Redis
}

type Redis struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type Database struct {
	RDB *redis.Client
}

type Data struct {
	Path      string `json:"path"`
	Code      string `json:"code"`
	Language  string `json:"language"`
	Timestamp int64  `json:"timestamp"`
	Hash      string `json:"hash"`
}

func InitDB(config Config) (*Database, error) {
	// * initialize redis RDB
	RDB := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Redis.Host, config.Redis.Port),
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RDB.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect redis: %w", err)
	}

	DB = Database{
		RDB: RDB,
	}

	return &DB, nil
}

func (s *Database) Close() error {
	return s.RDB.Close()
}

func getMD5(path string) string {
	h := md5.Sum([]byte(path))
	return hex.EncodeToString(h[:])
}
