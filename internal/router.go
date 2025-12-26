package internal

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/handler"
	"github.com/pardnchiu/go-faas/internal/utils"
)

func InitRouter(ctList []string) error {
	port := utils.GetWithDefaultInt("HTTP_PORT", 8080)

	r := gin.Default()

	r.POST("/upload", handler.Upload)
	r.POST("/run/*targetPath", handler.Run)
	r.POST("/run-now", handler.RunNow)

	return r.Run(fmt.Sprintf(":%d", port))
}
