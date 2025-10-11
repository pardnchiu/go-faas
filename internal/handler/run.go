package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

var (
	ctIdx   uint64 = 0
	ctList         = []string{}
	langMap        = map[string]string{
		".py": "python",
		".js": "javascript",
		".ts": "typescript",
	}
	runtimeMap = map[string]string{
		"python":     "python3",
		"javascript": "node",
		"typescript": "tsx",
	}
)

func InitCTList(list []string) {
	ctList = list
}

func getCT() string {
	nextIdx := atomic.AddUint64(&ctIdx, 1) % uint64(len(ctList))
	return ctList[nextIdx]
}

func Run(c *gin.Context) {
	targetPath := c.Param("targetPath")
	targetPath = strings.TrimPrefix(targetPath, "/")

	// TODO: use database to manage scripts
	log.Printf("Run script: %s", targetPath)
	if strings.Contains(targetPath, "..") {
		c.String(http.StatusBadRequest, "Invalid path")
		return
	}
	targetPath = filepath.Join("script", targetPath)

	// * directly pass request body to script
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 10<<20)
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
	ctPath := filepath.Join("/app", path)
	ctName := getCT()

	cmd := exec.Command("docker", "exec", ctName, runtime, ctPath, input)

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
