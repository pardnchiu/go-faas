package sandbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pardnchiu/go-faas/internal/utils"
)

var (
	extMap = map[string]string{
		"python":     ".py",
		"javascript": ".js",
		"typescript": ".ts",
	}
	runtimeMap = map[string]string{
		"python":     "python3",
		"javascript": "node",
		"typescript": "tsx",
	}
)

func SandboxCommand(ctx context.Context, lang string) (*exec.Cmd, error) {
	runtime := runtimeMap[lang]
	ext := extMap[lang]

	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	wrapperPath := filepath.Join(wd, "internal", "resource", fmt.Sprintf("wrapper%s", ext))
	sandboxPath := fmt.Sprintf("/wrapper%s", ext)

	baseArgs := []string{
		// "bwrap",
		"--ro-bind", "/usr", "/usr",
		"--ro-bind", "/lib", "/lib",
		"--ro-bind", "/lib64", "/lib64",
		"--ro-bind", wrapperPath, sandboxPath,
		"--tmpfs", "/tmp",
		"--proc", "/proc",
		"--dev", "/dev",
		"--unshare-all",
		"--unshare-net",
		"--die-with-parent",
		"--new-session",
		"--cap-drop", "ALL",
		"--chdir", "/tmp",
		"--tmpfs", "/home/sandbox",
		"--setenv", "HOME", "/home/sandbox",
		"--setenv", "PATH", "/usr/local/bin:/usr/bin:/bin",
		"--setenv", "TMPDIR", "/tmp",
		"--setenv", "LANG", "C.UTF-8",
		"--unsetenv", "LD_PRELOAD",
		"--unsetenv", "LD_LIBRARY_PATH",
	}

	if lang == "typescript" {
		nodeModulesPath := filepath.Join(wd, "node_modules")

		baseArgs = append(baseArgs,
			"--ro-bind", wd, wd,
			"--setenv", "NODE_PATH", nodeModulesPath,
		)
	}

	baseArgs = append(baseArgs, "--")

	if lang == "python" {
		baseArgs = append(baseArgs, runtime, "-u", sandboxPath)
	} else {
		baseArgs = append(baseArgs, runtime, sandboxPath)
	}

	maxCPU := utils.GetWithDefaultInt("MAX_CPUS", 1)
	maxMemory := utils.GetWithDefault("MAX_MEMORY", "128M")

	args := []string{
		"--scope", "--user", "--quiet",
		"-p", fmt.Sprintf("CPUQuota=%d%%", maxCPU*100),
		"-p", fmt.Sprintf("MemoryMax=%s", maxMemory),
		"-p", "MemorySwapMax=0",
		"--",
		"bwrap",
	}
	args = append(args, baseArgs...)

	return exec.CommandContext(ctx, "systemd-run", args...), nil
}
