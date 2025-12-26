package container

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	tickerPeriod          = 30 * time.Second
	checkHealthTimeout    = 5 * time.Second
	removeFromPoolTimeout = 100 * time.Millisecond
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

			if isHealth(ctName) {
				return
			}

			if !markUnhealthy(ctName) {
				slog.Debug("already marked: unhealthy",
					slog.String("container", ctName),
				)
				return
			}

			removeFromPool(ctName)

			if !markRebuilding(ctName) {
				slog.Warn("already in progress: rebuild",
					slog.String("container", ctName),
				)
				return
			}

			build(ctName)
		}(e)
	}

	wg.Wait()
}

func isHealth(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), checkHealthTimeout)
	defer cancel()

	// * check container is running
	cmd := exec.CommandContext(ctx, "podman", "inspect",
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

	// * not running
	if strings.TrimSpace(string(output)) != "true" {
		slog.Warn("container not running",
			slog.String("container", name),
		)
		return false
	}

	ctx, cancel = context.WithTimeout(context.Background(), checkHealthTimeout)
	defer cancel()

	// * check container health status
	cmd = exec.CommandContext(ctx, "podman", "inspect",
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
	isHealth := status == "no-healthcheck" || status == "healthy" || status == "starting"
	if !isHealth {
		slog.Warn("container is unhealthy",
			slog.String("container", name),
			slog.String("status", status),
		)
	}

	return true
}

func markUnhealthy(name string) bool {
	ctStatesMu.RLock()
	info, exists := ctStates[name]
	ctStatesMu.RUnlock()

	if !exists {
		return false
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	if info.State == StateIdle || info.State == StateAcquired {
		info.State = StateUnhealthy
		return true
	}

	return false
}

func removeFromPool(name string) {
	size := cap(ctPool)
	idx := 0

	for idx < size {
		select {
		case ct := <-ctPool:
			if ct != name {
				// * try put back to pool, if the pool is not full, and not to wait
				select {
				case ctPool <- ct:
				default:
				}
			}
			idx++
		case <-time.After(removeFromPoolTimeout):
			slog.Warn("timeout: removing from pool",
				slog.String("container", name),
			)
			return
		}
	}
}

func markRebuilding(name string) bool {
	ctStatesMu.RLock()
	info, exists := ctStates[name]
	ctStatesMu.RUnlock()

	if !exists {
		return false
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	if info.State != StateUnhealthy {
		return false
	}

	info.State = StateRebuilding
	return true
}
