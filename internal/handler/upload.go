package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/database"
)

type UploadRequest struct {
	Path     string `json:"path" binding:"required"`
	Code     string `json:"code" binding:"required"`
	Language string `json:"language" binding:"required"`
}

func Upload(c *gin.Context) {
	var req UploadRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "Invalid request payload")
		return
	}

	if strings.Contains(req.Path, "..") {
		c.String(http.StatusBadRequest, "Invalid path")
		return
	}

	if req.Language != "python" && req.Language != "javascript" && req.Language != "typescript" {
		c.String(http.StatusBadRequest, "Invalid path")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := database.DB.Add(ctx, database.Script{
		Path:     req.Path,
		Code:     req.Code,
		Language: req.Language,
	})

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
		"version":  version,
	})
}
