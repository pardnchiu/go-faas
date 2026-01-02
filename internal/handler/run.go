package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/container"
	"github.com/pardnchiu/go-faas/internal/database"
	"github.com/pardnchiu/go-faas/internal/utils"
)

type RunBody struct {
	Input  string `json:"input"`
	Stream bool   `json:"stream"`
}

type RunNowBody struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Input    string `json:"input"`
	Stream   bool   `json:"stream"`
}

var (
	timeoutRedis   = 5 * time.Second
	timeoutScript  time.Duration
	timeoutRequest time.Duration
	codeMaxSize    int64
	extMap         = map[string]string{
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

	if codeMaxSize == 0 {
		codeMaxSize = int64(utils.GetWithDefaultInt("CODE_MAX_SIZE", 256<<10))
	}

	var body RunBody

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, codeMaxSize)
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest,
			fmt.Sprintf("bad request: %s", err.Error()),
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutRedis)
	defer cancel()

	script, err := database.DB.Get(ctx, targetPath, version)
	if err != nil {
		c.String(http.StatusNotFound,
			fmt.Sprintf("bad request: %s", err.Error()),
		)
		return
	}

	if body.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError,
				"streaming unsupported",
			)
			return
		}

		ctx := c.Request.Context()

		flusher.Flush()

		res, err := runScriptWithSSE(script.Code, script.Language, body.Input, c.Writer, flusher, ctx)
		if err != nil {
			sendDone(c.Writer, flusher, "error", strings.ReplaceAll(err.Error(), "\n", " "))
			return
		}
		sendDone(c.Writer, flusher, "result", strings.ReplaceAll(res, "\n", " "))
		return
	}

	output, err := runScript(script.Code, script.Language, body.Input)
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("failed to run: %s", err.Error()),
		)
		return
	}

	sendResult(c, output)
}

func RunNow(c *gin.Context) {
	if codeMaxSize == 0 {
		codeMaxSize = int64(utils.GetWithDefaultInt("CODE_MAX_SIZE", 256<<10))
	}

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

	if body.Stream {
		c.Writer.Header().Set("Content-Type", "text/event-stream")
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")

		flusher, ok := c.Writer.(http.Flusher)
		if !ok {
			c.String(http.StatusInternalServerError,
				"streaming unsupported",
			)
			return
		}

		ctx := c.Request.Context()

		flusher.Flush()

		res, err := runScriptWithSSE(body.Code, body.Language, body.Input, c.Writer, flusher, ctx)
		if err != nil {
			sendDone(c.Writer, flusher, "error", strings.ReplaceAll(err.Error(), "\n", " "))
			return
		}
		sendDone(c.Writer, flusher, "result", strings.ReplaceAll(res, "\n", " "))
		return
	}

	output, err := runScript(body.Code, body.Language, body.Input)
	if err != nil {
		c.String(http.StatusInternalServerError,
			fmt.Sprintf("failed to run: %s", err.Error()),
		)
		return
	}

	sendResult(c, output)
}

func prepareScript(code, lang string) (string, string, string, error) {
	if timeoutScript == 0 {
		timeoutScript = time.Duration(utils.GetWithDefaultInt("TIMEOUT_SCRIPT", 30)) * time.Second
		timeoutRequest = timeoutScript + timeoutRedis
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutScript)
	defer cancel()

	ct, err := container.Get(ctx)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to get container: %w", err)
	}

	runtime := runtimeMap[lang]
	ext := extMap[lang]
	wrapPath := fmt.Sprintf("/app/wrapper%s", ext)

	return ct, runtime, wrapPath, nil
}

func runScript(code, lang, input string) (string, error) {
	ct, runtime, wrapPath, err := prepareScript(code, lang)
	defer func() {
		container.Release(ct)
	}()
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeoutRequest)
	defer cancel()

	// * prepare stdin with JSON containing code and input
	payload := map[string]string{
		"code":  code,
		"input": input,
	}
	payloadBody, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	cmd := exec.CommandContext(ctx, "podman",
		"exec", "-i", ct, runtime, wrapPath,
	)
	cmd.Stdin = strings.NewReader(string(payloadBody))

	output, err := cmd.CombinedOutput()
	if err != nil {
		// * timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("execution timeout (max %v)", timeoutRequest)
		}
		return "", fmt.Errorf("%s: %s", err, string(output))
	}

	raw := strings.TrimSpace(string(output))
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

func sendResult(c *gin.Context, output string) {
	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		switch v := data.(type) {
		case string:
			c.JSON(http.StatusOK, gin.H{
				"data": data,
				"type": "string",
			})
		case float64, int, int64, json.Number:
			c.JSON(http.StatusOK, gin.H{
				"data": v,
				"type": "number",
			})
		default:
			c.JSON(http.StatusOK, gin.H{
				"data": data,
				"type": "json",
			})
		}
	} else {
		c.JSON(http.StatusOK, gin.H{
			"data": output,
			"type": "text",
		})
	}
}
