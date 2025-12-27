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

type RunNowBody struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Input    string `json:"input"`
}

var (
	redisTimeout           = 5 * time.Second
	runScriptTimeout       = 25 * time.Second
	requestTimeout         = 30 * time.Second
	codeMaxSize      int64 = 256 << 10 // 256 KB
	extMap                 = map[string]string{
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
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, codeMaxSize)
	reqBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.String(http.StatusBadRequest,
			fmt.Sprintf("bad request: %s", err.Error()),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), redisTimeout)
	defer cancel()

	script, err := database.DB.Get(ctx, targetPath, version)
	if err != nil {
		c.String(http.StatusNotFound,
			fmt.Sprintf("bad request: %s", err.Error()),
		)
		return
	}

	output, err := runScript(script.Code, script.Language, string(reqBody))
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("failed to run: %s", err.Error()),
		)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		switch v := data.(type) {
		case string:
			c.JSON(http.StatusOK, gin.H{
				"output": v,
				"type":   "string",
			})
		case float64, int, int64, json.Number:
			c.JSON(http.StatusOK, gin.H{
				"output": v,
				"type":   "number",
			})
		default:
			c.JSON(http.StatusOK, gin.H{
				"output": data,
				"type":   "json",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": output,
		"type":   "text",
	})
}

func RunNow(c *gin.Context) {
	var body RunNowBody

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, codeMaxSize)

	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest,
			fmt.Sprintf("bad request: %s", err.Error()),
		)
		return
	}

	if _, ok := runtimeMap[body.Language]; !ok {
		c.String(http.StatusBadRequest,
			"bad request: unsupported language",
		)
		return
	}

	if strings.TrimSpace(body.Code) == "" {
		c.String(http.StatusBadRequest,
			"bad request: code is required",
		)
		return
	}

	slog.Info("run-now request",
		slog.String("language", body.Language),
		slog.Int("code_size", len(body.Code)),
		slog.Int("input_size", len(body.Input)),
	)
	fmt.Printf("Code:\n")
	fmt.Printf("%s\n\n", body.Code)
	if strings.TrimSpace(body.Input) != "" {
		fmt.Printf("Input:\n")
		fmt.Printf("%s\n", body.Input)
	}

	output, err := runScript(body.Code, body.Language, body.Input)
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("failed to run: %s", err.Error()),
		)
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		switch v := data.(type) {
		case string:
			c.JSON(http.StatusOK, gin.H{
				"output": v,
				"type":   "string",
			})
		case float64, int, int64, json.Number:
			c.JSON(http.StatusOK, gin.H{
				"output": v,
				"type":   "number",
			})
		default:
			c.JSON(http.StatusOK, gin.H{
				"output": data,
				"type":   "json",
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": output,
		"type":   "text",
	})
}

func runScript(code, lang, input string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runScriptTimeout)
	defer cancel()

	ct, err := container.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get container: %w", err)
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

	// * add (30 - 5)s timeout context
	ctx, cancel = context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "podman",
		"exec", "-i", ct, runtime, wrapPath, ctPath,
	)
	cmd.Stdin = strings.NewReader(input)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// * timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("execution timeout (max %v)", requestTimeout)
		}
		return "", fmt.Errorf("%s: %s", err, string(output))
	}

	raw := strings.TrimSpace(string(output))
	// try to find last valid JSON line (wrapper prints final return as JSON)
	if raw != "" {
		lines := strings.Split(raw, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			l := strings.TrimSpace(lines[i])
			if l == "" {
				continue
			}
			if json.Valid([]byte(l)) {
				return l, nil
			}
		}
	}

	// fallback: clean and return full output
	result := cleanOutput(raw)
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
