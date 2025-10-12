package docker

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	tickerPeriod = 30 * time.Second
	checkTimeout = 5 * time.Second
)

func healthCheck(list []string) {
	ticker := time.NewTicker(tickerPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-stopChannel:
			return
		case <-ticker.C:
			check(list)
		}
	}
}

func check(list []string) {
	var wg sync.WaitGroup

	for _, e := range list {
		wg.Add(1)

		go func(ctName string) {
			defer wg.Done()

			if !isHealth(ctName) {
				remove(ctName)
				rebuild(ctName)
			}
		}(e)
	}

	wg.Wait()
}

func isHealth(name string) bool {
	ctx1, cancel1 := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel1()

	// * first: check container is running or not
	cmd := exec.CommandContext(ctx1, "docker", "inspect",
		"--format", "{{.State.Running}}",
		name,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to check state",
			slog.String("container", name),
			slog.String("error", err.Error()),
		)
		return false
	}

	// * not running, return false
	isRunning := strings.TrimSpace(string(output)) == "true"
	if !isRunning {
		slog.Warn("container not running",
			slog.String("container", name),
		)
		return false
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), checkTimeout)
	defer cancel2()

	// * second: check container health status
	cmd = exec.CommandContext(ctx2, "docker", "inspect",
		"--format", "{{if .State.Health}}{{.State.Health.Status}}{{else}}no-healthcheck{{end}}",
		name,
	)
	output, err = cmd.CombinedOutput()
	if err != nil {
		slog.Error("failed to check health",
			slog.String("container", name),
			slog.String("error", err.Error()),
		)
		return false
	}

	status := strings.TrimSpace(string(output))
	isHealth := status == "no-healthcheck" ||
		status == "healthy" ||
		status == "starting"

	if !isHealth {
		slog.Warn("unhealthy",
			slog.String("container", name),
			slog.String("status", status),
		)
	}

	return isHealth
}
