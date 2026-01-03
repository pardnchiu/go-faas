package container

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"

	"github.com/pardnchiu/go-faas/internal/utils"
)

type ContainerState int

const (
	StateIdle ContainerState = iota
	StateAcquired
	StateExecuting
	StateUnhealthy
	StateRebuilding
)

type ContainerInfo struct {
	Name  string
	State ContainerState
	mu    sync.Mutex
}

func (s ContainerState) String() string {
	return [...]string{"Idle", "Acquired", "Executing", "Unhealthy", "Rebuilding"}[s]
}

var (
	ctPool              chan string
	ctStates            map[string]*ContainerInfo
	ctStatesMu          sync.RWMutex
	stopChannel         chan struct{}
	getContainerTimeout = 5 * time.Second
)

func Init() ([]string, error) {
	cores := runtime.NumCPU()
	if cores < 1 {
		cores = 1
	}
	maxContainers := utils.GetWithDefaultInt("MAX_CONTAINERS", cores*2)

	ctList := make([]string, maxContainers)
	for i := 0; i < maxContainers; i++ {
		ctList[i] = fmt.Sprintf("go-faas-runtime-%d", i)
	}

	if err := start(ctList); err != nil {
		return nil, fmt.Errorf("failed to start containers: %w", err)
	}

	ctPool = make(chan string, len(ctList))
	ctStates = make(map[string]*ContainerInfo, len(ctList))
	stopChannel = make(chan struct{})

	for _, name := range ctList {
		ctStates[name] = &ContainerInfo{
			Name:  name,
			State: StateIdle,
		}
		ctPool <- name
	}

	go checkTimer(ctList)

	return ctList, nil
}

func Get(ctx context.Context) (string, error) {
	var cancel context.CancelFunc
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, getContainerTimeout)
		defer cancel()
	}
	select {
	case name := <-ctPool:
		if err := getContainer(name); err != nil {
			select {
			case ctPool <- name:
			default:
			}
			return "", fmt.Errorf("container in invalid state: %w", err)
		}
		return name, nil
	case <-ctx.Done():
		return "", fmt.Errorf("acquire timeout: %w", ctx.Err())
	}
}

func getContainer(name string) error {
	ctStatesMu.RLock()
	info, exists := ctStates[name]
	ctStatesMu.RUnlock()

	if !exists {
		return fmt.Errorf("container not found: %s", name)
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	if info.State != StateIdle {
		return fmt.Errorf("container not idle: %v", info.State)
	}

	info.State = StateAcquired
	return nil
}

func Release(name string) {
	ctStatesMu.RLock()
	info, exists := ctStates[name]
	ctStatesMu.RUnlock()

	if !exists {
		slog.Error("release unknown container",
			slog.String("container", name),
		)
		return
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	if info.State == StateUnhealthy || info.State == StateRebuilding {
		return
	}

	info.State = StateIdle

	select {
	case ctPool <- name:
	default:
		slog.Warn("pool full when releasing",
			slog.String("container", name),
		)
	}
}
