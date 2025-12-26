package container

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func start(list []string) error {
	// * folder real path
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	folderPath := filepath.Join(wd, "temp")

	gpuEnabled := os.Getenv("GPU_ENABLED") == "true"
	dockerfile := "Dockerfile.runtime"
	if gpuEnabled {
		dockerfile = "Dockerfile.runtime.gpu"
	}

	cmd := exec.Command("podman", "build",
		"-t", "faas-runtime",
		"-f", dockerfile,
		".",
	)
	cmd.Dir = wd
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to build runtime: %s: %w", string(output), err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	ch := make(chan error, 1)

	for _, e := range list {
		wg.Add(1)
		go func(ctName string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			exec.Command("podman", "stop", ctName).Run()
			exec.Command("podman", "rm", ctName).Run()

			runArgs := []string{
				"run",
				"-d",
				"--name", ctName,
				"-v", fmt.Sprintf("%s:/app/temp:Z", folderPath),
				"--health-cmd", "test -d /app/temp || exit 1",
				"--health-interval", "10s",
				"--health-timeout", "5s",
				"--health-retries", "3",
			}

			if gpuEnabled {
				runArgs = append(runArgs,
					"--device", "nvidia.com/gpu=all",
					"--security-opt", "label=disable",
				)
			}

			runArgs = append(runArgs, "faas-runtime")

			wgCmd := exec.Command("podman", runArgs...)
			if output, err := wgCmd.CombinedOutput(); err != nil {
				select {
				case ch <- fmt.Errorf("failed to start %s: %s: %w", ctName, string(output), err):
					cancel()
				default:
				}
				return
			}
		}(e)
	}

	wg.Wait()
	close(ch)

	if err := <-ch; err != nil {
		return err
	}

	return nil
}

func Stop(list []string) {
	slog.Info("Waiting for stopping containers")
	close(stopChannel)

	var wg sync.WaitGroup

	for _, name := range list {
		wg.Add(1)
		go func(containerName string) {
			defer wg.Done()
			exec.Command("podman", "stop", containerName).Run()
			exec.Command("podman", "rm", containerName).Run()
		}(name)
	}

	wg.Wait()
	os.Exit(0)
}
