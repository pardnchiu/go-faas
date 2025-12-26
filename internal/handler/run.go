package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/container"
	"github.com/pardnchiu/go-faas/internal/database"
)

var (
	redisTimeout        = 5 * time.Second
	scriptTimeout       = 30 * time.Second
	scriptMax     int64 = 10 << 20
	extMap              = map[string]string{
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

func Run(c *gin.Context) {
	targetPath := c.Param("targetPath")
	targetPath = strings.TrimPrefix(targetPath, "/")

	queryVersion := c.Query("version")
	var version int64
	if queryVersion != "" {
		// * version invalid, use latest
		v, err := strconv.ParseInt(queryVersion, 10, 64)
		if err == nil {
			version = v
		}
	}

	// * directly pass request body to script
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, scriptMax)
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest, "Failed to read request body")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()

	scriptData, err := database.DB.Get(ctx, targetPath, version)
	if err != nil {
		slog.Warn("function not found in redis",
			slog.String("path", targetPath),
			slog.String("error", err.Error()),
		)
		c.String(http.StatusNotFound, "Function not found")
		return
	}

	slog.Info("run function",
		slog.String("path", targetPath),
		slog.String("language", scriptData.Language),
		slog.Int64("version", scriptData.Timestamp),
	)

	// * change to use directly code
	output, err := runScript(scriptData.Code, scriptData.Language, string(reqBody))
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to run script: %s", err.Error()))
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

func runScript(code, lang, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ct, err := container.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to acquire container: %w", err)
	}
	defer container.Release(ct)

	runtime := runtimeMap[lang]
	ext := extMap[lang]

	tempFile := fmt.Sprintf("temp_%d%s", time.Now().UnixNano(), ext)
	localPath := filepath.Join("temp", tempFile)

	if err := os.WriteFile(localPath, []byte(code), 0644); err != nil {
		return "", fmt.Errorf("failed to write code: %w", err)
	}
	defer func() {
		if err := os.Remove(localPath); err != nil {
			slog.Warn("failed to cleanup temp file",
				slog.String("file", localPath),
				slog.String("error", err.Error()),
			)
		}
	}()

	wrapPath := fmt.Sprintf("/app/wrapper%s", ext)
	ctPath := filepath.Join("/app/temp", tempFile)

	// * add 30s timeout context
	ctx, cancel = context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman", "exec", "-i", ct, runtime, wrapPath, ctPath)
	cmd.Stdin = strings.NewReader(input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// * timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("execution timeout (max %v)", runNowTimeout)
		}
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
