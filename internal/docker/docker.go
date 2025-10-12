package docker

import (
	"fmt"
	"log/slog"
	"runtime"
	"sync"
	"time"
)

var (
	ctMutex     sync.RWMutex
	ctPool      chan string
	stopChannel chan struct{}
)

func InitDocker() ([]string, error) {
	cores := runtime.NumCPU()
	if cores < 1 {
		cores = 1
	}

	ctList := make([]string, cores*2)
	for i := 0; i < cores*2; i++ {
		ctList[i] = fmt.Sprintf("go-faas-runtime-%d", i)
	}

	if err := start(ctList); err != nil {
		return nil, fmt.Errorf("[InitDocker: %v]", err)
	}

	ctPool = make(chan string, len(ctList))
	for _, e := range ctList {
		ctPool <- e
	}

	stopChannel = make(chan struct{})
	go healthCheck(ctList)

	return ctList, nil
}

func Get() string {
	return <-ctPool
}

func Release(name string) {
	ctPool <- name
}

func remove(name string) {
	timeout := time.After(100 * time.Millisecond)
	size := cap(ctPool)
	idx := 0

	for idx < size {
		select {
		case ct := <-ctPool:
			if ct == name {
				return
			}
			ctPool <- ct
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
