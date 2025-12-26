package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	runNowTimeout       = 30 * time.Second
	runNowMaxSize int64 = 5 << 20
)

type RunNowRequest struct {
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
	Input    string `json:"input"`
}

func RunNow(c *gin.Context) {
	var req RunNowRequest

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, runNowMaxSize)

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request payload",
		})
		return
	}

	if _, ok := runtimeMap[req.Language]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Unsupported language: %s", req.Language),
		})
		return
	}

	if len(req.Code) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Code cannot be empty",
		})
		return
	}

	slog.Info("run-now request",
		slog.String("language", req.Language),
		slog.Int("code_size", len(req.Code)),
		slog.Int("input_size", len(req.Input)),
	)

	output, err := runScript(req.Code, req.Language, req.Input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Execution failed: %s", err.Error()),
		})
		return
	}

	var data interface{}
	if err := json.Unmarshal([]byte(output), &data); err == nil {
		c.JSON(http.StatusOK, gin.H{
			"output": data,
			"type":   "json",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"output": output,
		"type":   "text",
	})
}
