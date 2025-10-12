package docker

import (
	"log/slog"
	"os"
	"os/exec"
	"sync"
)

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
