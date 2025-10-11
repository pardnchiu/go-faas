package internal

import (
	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/handler"
)

func InitRouter() error {
	r := gin.Default()

	r.POST("/run/*targetPath", handler.Run)

	return r.Run(":8080")
}
