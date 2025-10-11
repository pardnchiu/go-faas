package docker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

const (
	ctMax        = 5
	tickerPeriod = 30 * time.Second
	checkTimeout = 5 * time.Second
)

var (
	ctMutex     sync.RWMutex
	ctPool      chan string
	stopChannel chan struct{}
)

func InitDocker() ([]string, error) {
	ctList := make([]string, ctMax)
	for i := 0; i < ctMax; i++ {
		ctList[i] = fmt.Sprintf("go-faas-runtime-%d", i)
	}

	if err := start(ctList); err != nil {
		return nil, fmt.Errorf("[InitDocker: %v]", err)
	}

	initPool(ctList)

	stopChannel = make(chan struct{})
	go healthCheck(ctList)

	return ctList, nil
}

func initPool(list []string) {
	ctPool = make(chan string, len(list))
	for _, e := range list {
		ctPool <- e
	}
}

func Get() string {
	return <-ctPool
}

func Release(name string) {
	ctPool <- name
}

func start(list []string) error {
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
				"-v", fmt.Sprintf("%s:/app/script", folderPath),
				"--health-cmd", "test -d /app/script || exit 1",
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

func Stop(list []string) {
	slog.Info("Waiting for stopping containers")
	close(stopChannel)

	var wg sync.WaitGroup

	for _, name := range list {
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
	folderPath := filepath.Join(wd, "script")

	exec.Command("docker", "stop", name).Run()
	exec.Command("docker", "rm", name).Run()

	cmd := exec.Command("docker", "run",
		"-d",
		"--name", name,
		"-v", fmt.Sprintf("%s:/app/script", folderPath),
		"--health-cmd", "test -d /app/script || exit 1",
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

func remove(name string) {
	timeout := time.After(100 * time.Millisecond)
	size := cap(ctPool)
	idx := 0

	for idx < size {
		select {
		case ct := <-ctPool:
			if ct != name {
				ctPool <- ct
			} else {
				return
			}
			idx++
		case <-timeout:
			slog.Warn("timeout at removing container",
				slog.String("container", name),
			)
			return
		}
	}
}

func add(name string) {
	select {
	case ctPool <- name:
		break
	default:
		slog.Warn("pool is max",
			slog.String("container", name),
		)
	}
}
