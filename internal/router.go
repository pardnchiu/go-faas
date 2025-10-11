package internal

import (
	"github.com/gin-gonic/gin"
)

func InitRouter() error {
	r := gin.Default()

	return r.Run(":8080")
}
