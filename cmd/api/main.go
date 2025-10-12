package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pardnchiu/go-faas/internal"
	"github.com/pardnchiu/go-faas/internal/database"
	"github.com/pardnchiu/go-faas/internal/docker"
)

func main() {
	// * initialize database
	db, err := database.InitDB(database.Config{
		Redis: &database.Redis{
			Host:     "localhost",
			Port:     6379,
			Password: "0123456789",
			DB:       0,
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
