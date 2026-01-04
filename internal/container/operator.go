package container

// import (
// 	"context"
// 	"fmt"
// 	"log/slog"
// 	"os"
// 	"os/exec"
// 	"sync"

// 	"github.com/pardnchiu/go-faas/internal/utils"
// )

// func start(list []string) error {
// 	// * folder real path
// 	wd, err := os.Getwd()
// 	if err != nil {
// 		return fmt.Errorf("failed to get working directory: %w", err)
// 	}

// 	gpuEnabled := utils.GetWithDefault("GPU_ENABLED", "false") == "true"
// 	dockerfile := "Dockerfile.runtime"
// 	if gpuEnabled {
// 		dockerfile = "Dockerfile.runtime.gpu"
// 	}

// 	cmd := exec.Command("podman", "build",
// 		"-t", "faas-runtime",
// 		"-f", dockerfile,
// 		".",
// 	)
// 	cmd.Dir = wd
// 	if output, err := cmd.CombinedOutput(); err != nil {
// 		return fmt.Errorf("failed to build runtime: %s: %w", string(output), err)
// 	}

// 	ctx, cancel := context.WithCancel(context.Background())
// 	defer cancel()

// 	var wg sync.WaitGroup
// 	ch := make(chan error, 1)

// 	for _, e := range list {
// 		wg.Add(1)

// 		go func(ctName string) {
// 			defer wg.Done()

// 			select {
// 			case <-ctx.Done():
// 				return
// 			default:
// 			}

// 			exec.Command("podman", "stop", ctName).Run()
// 			exec.Command("podman", "rm", ctName).Run()

// 			args := argsForRun(ctName)
// 			wgCmd := exec.Command("podman", args...)
// 			if output, err := wgCmd.CombinedOutput(); err != nil {
// 				select {
// 				case ch <- fmt.Errorf("failed to start %s: %s: %w", ctName, string(output), err):
// 					cancel()
// 				default:
// 				}
// 				return
// 			}
// 		}(e)
// 	}

// 	wg.Wait()
// 	close(ch)

// 	if err := <-ch; err != nil {
// 		return err
// 	}

// 	return nil
// }

// func Stop(list []string) {
// 	slog.Info("waiting for stopping containers")

// 	close(stopChannel)

// 	var wg sync.WaitGroup

// 	for _, name := range list {
// 		wg.Add(1)

// 		go func(ctName string) {
// 			defer wg.Done()
// 			exec.Command("podman", "stop", ctName).Run()
// 			exec.Command("podman", "rm", ctName).Run()
// 		}(name)
// 	}

// 	wg.Wait()
// 	os.Exit(0)
// }
