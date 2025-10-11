package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

var langMap = map[string]string{
	".py": "python",
	".js": "javascript",
	".ts": "typescript",
}
var runtimeMap = map[string]string{
	"python":     "python3",
	"javascript": "node",
	"typescript": "tsx",
}

func Run(c *gin.Context) {
	targetPath := c.Param("targetPath")
	targetPath = strings.TrimPrefix(targetPath, "/")

	// TODO: use database to manage scripts
	targetPath = filepath.Join("script", targetPath)
	if strings.Contains(targetPath, "..") {
		c.String(http.StatusBadRequest, "Invalid path")
		return
	}
	targetPath, err := filepath.Abs(targetPath)
	if err != nil {
		c.String(http.StatusBadRequest, "Failed to get absolute path")
		return
	}
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		c.String(http.StatusNotFound, "Failed to find the script")
		return
	}
	// * directly pass request body to script
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20) // 10MB
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "Failed to read request body")
		return
	}

	// TODO: check by database record
	ext := strings.ToLower(filepath.Ext(targetPath))
	lang := langMap[ext]
	if lang == "" {
		c.String(http.StatusBadRequest, "Unsupported file type")
		return
	}

	// * run script
	output, err := runScript(targetPath, lang, string(reqBody))
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		// * output can parse, return JSON type
		c.JSON(http.StatusOK, data)
		return
	}

	c.String(http.StatusOK, output)
}

func runScript(path, lang, input string) (string, error) {
	runtime := runtimeMap[lang]
	if _, err := exec.LookPath(runtime); err != nil {
		return "", fmt.Errorf("%s not found", runtime)
	}
	cmd := exec.Command(runtime, path, input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, string(output))
	}

	result := strings.TrimSpace(string(output))
	result = cleanOutput(result)

	return result, nil
}

func cleanOutput(output string) string {
	rowList := strings.Split(output, "\n")
	var newList []string

	for _, row := range rowList {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}

		if strings.Contains(row, "Warning:") ||
			strings.Contains(row, "MODULE_TYPELESS_PACKAGE_JSON") ||
			strings.Contains(row, "Use `node --trace-warnings") ||
			strings.Contains(row, "ExperimentalWarning") {
			continue
		}

		newList = append(newList, row)
	}

	return strings.Join(newList, "\n")
}
