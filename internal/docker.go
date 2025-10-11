package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

const (
	ctMax = 5
)

func InitDocker() ([]string, error) {
	ctList := make([]string, ctMax)
	for i := 0; i < ctMax; i++ {
		ctList[i] = fmt.Sprintf("go-faas-runtime-%d", i)
	}

	if err := startContainer(ctList); err != nil {
		return nil, fmt.Errorf("[InitDocker: %v]", err)
	}

	return ctList, nil
}

func startContainer(ctList []string) error {
	// * folder real path
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("[startContainer-0: %v]", err)
	}
	folderPath := filepath.Join(wd, "script")

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
	for _, e := range ctList {
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
				"-v", fmt.Sprintf("%s:/app/script", folderPath),
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

func StopContainer(ctList []string) {
	log.Println("waiting for stopping containers")

	var wg sync.WaitGroup

	for _, name := range ctList {
		wg.Add(1)
		go func(containerName string) {
			defer wg.Done()
			exec.Command("docker", "stop", containerName).Run()
			exec.Command("docker", "rm", containerName).Run()
		}(name)
	}

	wg.Wait()
	os.Exit(0)
}
