package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/database"
)

func Upload(c *gin.Context) {
	var req struct {
		Path     string `json:"path" binding:"required"`
		Code     string `json:"code" binding:"required"`
		Language string `json:"language" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.Contains(req.Path, "..") {
		c.String(http.StatusBadRequest, "Invalid path")
		return
	}

	if _, ok := runtimeMap[req.Language]; !ok {
		c.String(http.StatusBadRequest, fmt.Sprintf("Unsupported language: %s", req.Language))
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	timestamp, err := database.DB.Add(ctx, req.Path, req.Code, req.Language)
	if err != nil {
		slog.Error("failed to save function",
			slog.String("error", err.Error()),
		)
		c.String(http.StatusInternalServerError, "Failed to save function")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"path":     req.Path,
		"language": req.Language,
		"version":  timestamp,
	})
}
