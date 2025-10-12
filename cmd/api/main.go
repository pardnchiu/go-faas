package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/pardnchiu/go-faas/internal"
	"github.com/pardnchiu/go-faas/internal/database"
	"github.com/pardnchiu/go-faas/internal/docker"
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Error("Error loading .env file",
			slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvWithDefaultInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}

func main() {
	host := getEnvWithDefault("REDIS_HOST", "localhost")
	port := getEnvWithDefaultInt("REDIS_PORT", 6379)
	password := getEnvWithDefault("REDIS_PASSWORD", "")
	dbNum := getEnvWithDefaultInt("REDIS_DB", 0)

	// * initialize db
	db, err := database.InitDB(database.Config{
		Redis: &database.Redis{
			Host:     host,
			Port:     port,
			Password: password,
			DB:       dbNum,
		},
	})
	if err != nil {
		slog.Error("Failed to initialize database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer db.Close()

	// * initialize 5 containers for running scripts and minus cold start time
	ctList, err := docker.InitDocker()
	if err != nil {
		slog.Error("Failed to initialize Docker", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer docker.Stop(ctList)

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)

	// * listen for signal to shutdown containers
	go func() {
		<-channel
		docker.Stop(ctList)
	}()

	// * initialize router and start server
	if err := internal.InitRouter(ctList); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
