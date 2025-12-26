package main

import (
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/pardnchiu/go-faas/internal"
	"github.com/pardnchiu/go-faas/internal/container"
	"github.com/pardnchiu/go-faas/internal/database"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, using environment variables")
	}

	// Initialize Redis database
	if err := database.Init(); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Initialize Docker/Podman container pool
	ctList, err := container.Init()
	if err != nil {
		log.Fatalf("Failed to initialize container pool: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		slog.Info("Received shutdown signal, cleaning up...")
		container.Stop(ctList)
	}()

	// Start HTTP server
	slog.Info("Starting FaaS service on :8080")
	if err := internal.InitRouter(ctList); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
