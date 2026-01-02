package container

import (
	"log/slog"
	"os/exec"
	"time"
)

var (
	addToPoolTimeout = 100 * time.Millisecond
)

func build(name string) {
	slog.Info("start to build container",
		slog.String("container", name),
	)

	// * Clean up container if it exists
	exec.Command("podman", "stop", name).Run()
	exec.Command("podman", "rm", name).Run()

	cmd := exec.Command("podman", "run",
		"-d",
		"--name", name,
		"faas-runtime",
	)
	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Error("failed to build",
			slog.String("container", name),
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		markIdle(name)
		return
	}

	markIdle(name)

	select {
	case ctPool <- name:
	case <-time.After(addToPoolTimeout):
		slog.Warn("timeout: add to pool",
			slog.String("container", name),
			slog.String("error", "pool full"),
		)
	}

	slog.Info("success to build container",
		slog.String("container", name),
	)
}

func markIdle(name string) {
	ctStatesMu.RLock()
	info, exists := ctStates[name]
	ctStatesMu.RUnlock()

	if !exists {
		return
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	info.State = StateIdle
}
