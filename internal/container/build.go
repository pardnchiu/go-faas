package container

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

var (
	addToPoolTimeout = 100 * time.Millisecond
)

func build(name string) {
	slog.Info("start to build container",
		slog.String("container", name),
	)

	wd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get working directory",
			slog.String("container", name),
			slog.String("error", err.Error()),
		)
		markIdle(name)
		return
	}
	folderPath := filepath.Join(wd, "temp")

	// * Clean up container if it exists
	exec.Command("podman", "stop", name).Run()
	exec.Command("podman", "rm", name).Run()

	cmd := exec.Command("podman", "run",
		"-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:/app/temp:Z", folderPath), // * :Z for SELinux context
		"--health-cmd", "test -d /app/temp || exit 1",
		"--health-interval", "10s",
		"--health-timeout", "5s",
		"--health-retries", "3",
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
