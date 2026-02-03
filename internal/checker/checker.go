package checker

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func getOSName() (string, error) {
	switch runtime.GOOS {
	case "linux":
		file, err := os.Open("/etc/os-release")
		if err != nil {
			return "", err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			parts := strings.SplitN(scanner.Text(), "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := parts[0]
			val := parts[1]

			if key == "" || val == "" {
				return "", fmt.Errorf("invalid line in /etc/os-release: %s", scanner.Text())
			}
			if key == "NAME" {
				return strings.Trim(val, `"`), nil
			}
		}

		return "", fmt.Errorf("NAME not found in /etc/os-release")
	default:
		return "", fmt.Errorf("unsupported os: %s", runtime.GOOS)
	}
}

func execCommand(pm string) (*exec.Cmd, error) {
	var cmd *exec.Cmd
	switch pm {
	case "apt", "apk":
		cmd = exec.Command("sudo", pm, "update")
	case "dnf":
		cmd = exec.Command("sudo", pm, "check-update")
	case "pacman":
		cmd = exec.Command("sudo", pm, "-Syu", "--noconfirm")
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pm)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 100 {
			// dnf check-update => 100: 表示有可用更新
		} else {
			return nil, fmt.Errorf("failed to run update command: %w", err)
		}
	}

	newArgs := []string{pm}
	switch pm {
	case "apt", "dnf":
		newArgs = append(newArgs, "install", "-y")
	case "pacman":
		newArgs = append(newArgs, "-S", "--noconfirm")
	case "apk":
		newArgs = append(newArgs, "add")
	default:
		return nil, fmt.Errorf("unsupported package manager: %s", pm)
	}
	newArgs = append(newArgs, "bubblewrap", "nodejs", "npm", "python3")
	cmd = exec.Command("sudo", newArgs...)

	return cmd, nil
}

func CheckPackage() error {
	var cmd *exec.Cmd
	var err error

	osName, err := getOSName()
	if err != nil {
		return err
	}

	if !isMissing() {
		return nil
	}

	switch osName {
	case "Ubuntu", "Debian":
		if _, err := os.Stat("/usr/bin/apt"); os.IsNotExist(err) {
			return fmt.Errorf("apt package manager not found")
		}
		cmd, err = execCommand("apt")
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
	case "Rocky Linux", "Alma Linux", "Fedora", "RedHat":
		if _, err := os.Stat("/usr/bin/dnf"); os.IsNotExist(err) {
			return fmt.Errorf("dnf package manager not found")
		}
		cmd, err = execCommand("dnf")
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
	case "Arch Linux":
		if _, err := os.Stat("/usr/bin/pacman"); os.IsNotExist(err) {
			return fmt.Errorf("pacman package manager not found")
		}
		cmd, err = execCommand("pacman")
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
	case "Alpine Linux":
		if _, err := os.Stat("/sbin/apk"); os.IsNotExist(err) {
			return fmt.Errorf("apk package manager not found")
		}
		cmd, err = execCommand("apk")
		if err != nil {
			return fmt.Errorf("failed to update package: %w", err)
		}
	default:
		return fmt.Errorf("unsupported os: %s", osName)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install package: %w", err)
	}

	return nil
}

func isMissing() bool {
	return !checkNodeInstalled() ||
		!checkTSInstalled() ||
		!checkEsbuildInstalled() ||
		!checkPythonInstalled()
}

func checkNodeInstalled() bool {
	_, err := exec.Command("node", "--version").Output()
	if err != nil {
		return false
	}
	return true
}

func checkTSInstalled() bool {
	_, err := exec.Command("tsc", "--version").Output()
	if err != nil {
		return false
	}
	return true
}

func checkEsbuildInstalled() bool {
	_, err := exec.Command("esbuild", "--version").Output()
	if err != nil {
		return false
	}
	return true
}

func checkPythonInstalled() bool {
	for _, cmd := range []string{"python3", "python"} {
		_, err := exec.Command(cmd, "--version").Output()
		if err != nil {
			continue
		}
		return true
	}
	return false
}
