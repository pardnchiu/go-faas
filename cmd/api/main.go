package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/pardnchiu/go-faas/internal"
)

func main() {
	// * initialize 5 containers for running scripts and minus cold start time
	ctList, err := internal.InitDocker()
	if err != nil {
		slog.Error("Failed to initialize Docker", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer internal.StopContainer(ctList)

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt, syscall.SIGTERM)

	// * listen for signal to shutdown containers
	go func() {
		<-channel
		internal.StopContainer(ctList)
	}()

	// * initialize router and start server
	if err := internal.InitRouter(ctList); err != nil {
		slog.Error("Failed to start server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}
