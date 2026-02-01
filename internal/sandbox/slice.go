package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pardnchiu/go-faas/internal/utils"
)

func NewSlice() error {
	maxCPU := utils.GetWithDefaultInt("MAX_CPUS", 1)
	maxMemory := utils.GetWithDefault("MAX_MEMORY", "128M")

	sliceContent := fmt.Sprintf(`[Unit]
Description=FaaS Sandbox

[Slice]
CPUQuota=%d%%
MemoryMax=%s
MemorySwapMax=0
`, maxCPU*100, maxMemory)

	folderPath := filepath.Join(os.Getenv("HOME"), ".config/systemd/user")
	os.MkdirAll(folderPath, 0755)

	path := filepath.Join(folderPath, "go-faas-slice")
	if err := os.WriteFile(path, []byte(sliceContent), 0644); err != nil {
		return err
	}

	exec.Command("systemctl", "--user", "daemon-reload").Run()
	exec.Command("systemctl", "--user", "start", "go-faas-slice").Run()

	return nil
}
