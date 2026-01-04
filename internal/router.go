package internal

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pardnchiu/go-faas/internal/handler"
	"github.com/pardnchiu/go-faas/internal/utils"
)

func CreateServer() *http.Server {
	port := utils.GetWithDefaultInt("HTTP_PORT", 8080)

	r := gin.Default()

	r.POST("/upload", handler.Upload)
	r.POST("/run/*targetPath", handler.Run)
	r.POST("/run-now", handler.RunNow)

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
}

func InitRouter() error {
	return CreateServer().ListenAndServe()
}
