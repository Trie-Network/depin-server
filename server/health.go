package server

import (
	"depin-server/utils"
	"os"

	"github.com/gin-gonic/gin"
)

func (s *DepinServer) HandleHealthCheck(c *gin.Context) {
	status := "NOOP"
	if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
		status = "OK"
	}
	utils.RespondSuccess(c, "Health check", gin.H{
		"status": status,
	})
}
