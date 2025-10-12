package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

func start(list []string) error {
	// * folder real path
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("[startContainer-0: %v]", err)
	}
	folderPath := filepath.Join(wd, "temp")

	cmd := exec.Command("docker", "build",
		"-t", "faas-runtime",
		"-f", "Dockerfile.runtime",
		".",
	)
	cmd.Dir = wd
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("[startContainer-1: %s: %s]", err, string(output))
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	channel := make(chan error, 1)
	for _, e := range list {
		wg.Add(1)
		go func(ctName string) {
			defer wg.Done()

			select {
			case <-ctx.Done():
				return
			default:
			}

			exec.Command("docker", "stop", ctName).Run()
			exec.Command("docker", "rm", ctName).Run()

			wgCmd := exec.Command("docker", "run",
				"-d",
				"--name", ctName,
				"-v", fmt.Sprintf("%s:/app/temp", folderPath),
				"--health-cmd", "test -d /app/temp || exit 1",
				"--health-interval", "10s",
				"--health-timeout", "5s",
				"--health-retries", "3",
				"faas-runtime",
			)
			if output, err := wgCmd.CombinedOutput(); err != nil {
				select {
				case channel <- fmt.Errorf("[startContainer-2: %s: %s: %s]", ctName, err, string(output)):
					cancel()
				default:
				}
				return
			}
		}(e)
	}

	wg.Wait()
	close(channel)

	if err := <-channel; err != nil {
		return err
	}

	return nil
}
