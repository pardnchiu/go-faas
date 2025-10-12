package docker

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
)

func rebuild(name string) {
	ctMutex.Lock()
	defer ctMutex.Unlock()

	wd, err := os.Getwd()
	if err != nil {
		slog.Error("failed to get real path",
			slog.String("error", err.Error()),
		)
		return
	}
	folderPath := filepath.Join(wd, "temp")

	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()

	cmd := exec.Command("docker", "run",
		"-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:/app/temp", folderPath),
		"--health-cmd", "test -d /app/temp || exit 1",
		"--health-interval", "10s",
		"--health-timeout", "5s",
		"--health-retries", "3",
		"faas-runtime",
	)

	if output, err := cmd.CombinedOutput(); err != nil {
		slog.Error("failed to rebuild",
			slog.String("container", name),
			slog.String("error", err.Error()),
			slog.String("output", string(output)),
		)
		return
	}

	add(name)
}
