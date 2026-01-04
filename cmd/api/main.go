package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/pardnchiu/go-faas/internal"
	"github.com/pardnchiu/go-faas/internal/database"
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("failed to find .env, using system environment variables")
	}
}

func main() {
	if err := database.Init(); err != nil {
		slog.Error("failed to initialize db", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Initialize Docker/Podman container pool
	// ctList, err := container.Init()
	// if err != nil {
	// 	log.Fatalf("Failed to initialize container pool: %v", err)
	// }

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// go func() {
	// 	<-sigChan
	// 	slog.Info("Received shutdown signal, cleaning up...")
	// 	// container.Stop(ctList)
	// }()

	// Start HTTP server
	// slog.Info("Starting FaaS service on :8080")
	// if err := internal.InitRouter(); err != nil {
	// 	log.Fatalf("Failed to start server: %v", err)
	// }
	srv := internal.CreateServer()

	go func() {
		slog.Info("start", "port", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("failed to start", "error", err)
			os.Exit(1)
		}
	}()

	<-sigChan
	slog.Info("Received shutdown signal, cleaning up...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}
}
