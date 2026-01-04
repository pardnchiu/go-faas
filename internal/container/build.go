package container

// import (
// 	"fmt"
// 	"log/slog"
// 	"os/exec"
// 	"time"

// 	"github.com/pardnchiu/go-faas/internal/utils"
// )

// var (
// 	addToPoolTimeout = 100 * time.Millisecond
// )

// func argsForRun(ctName string) []string {
// 	gpuEnabled := utils.GetWithDefault("GPU_ENABLED", "false") == "true"
// 	maxCPU := utils.GetWithDefaultFloat("MAX_CPUS_PER_CONTAINER", 1.0)
// 	maxMemory := utils.GetWithDefaultInt("MAX_MEMORY_PER_CONTAINER", 128<<20)
// 	args := []string{
// 		"run",
// 		"-d",
// 		"--name", ctName,
// 		"--cpus", fmt.Sprintf("%.2f", maxCPU),
// 		"--memory", fmt.Sprintf("%dm", maxMemory/(1<<20)),
// 		"--memory-swap", fmt.Sprintf("%dm", maxMemory/(1<<20)),
// 	}

// 	if gpuEnabled {
// 		args = append(args,
// 			"--device", "nvidia.com/gpu=all",
// 			"--security-opt", "label=disable",
// 		)
// 	}

// 	args = append(args, "faas-runtime")

// 	return args
// }

// func rebuild(ctName string) {
// 	slog.Info("start to build container",
// 		slog.String("container", ctName),
// 	)

// 	// * Clean up container if it exists
// 	// ? without error handling, because not necessary, if not exists, just continue
// 	exec.Command("podman", "stop", ctName).Run()
// 	exec.Command("podman", "rm", ctName).Run()

// 	args := argsForRun(ctName)
// 	cmd := exec.Command("podman", args...)
// 	if output, err := cmd.CombinedOutput(); err != nil {
// 		slog.Error("failed to build",
// 			slog.String("container", ctName),
// 			slog.String("error", err.Error()),
// 			slog.String("output", string(output)),
// 		)
// 		markIdle(ctName)
// 		return
// 	}

// 	markIdle(ctName)

// 	select {
// 	case ctPool <- ctName:
// 		slog.Info("added container to pool",
// 			slog.String("container", ctName),
// 		)
// 	case <-time.After(addToPoolTimeout):
// 		slog.Warn("timeout: add to pool",
// 			slog.String("container", ctName),
// 			slog.String("error", "pool full"),
// 		)
// 	}

// 	slog.Info("success to rebuild container",
// 		slog.String("container", ctName),
// 	)
// }

// func markIdle(name string) {
// 	ctStatesMu.RLock()
// 	info, exists := ctStates[name]
// 	ctStatesMu.RUnlock()

// 	if !exists {
// 		return
// 	}

// 	info.mu.Lock()
// 	defer info.mu.Unlock()

// 	info.State = StateIdle
// }
