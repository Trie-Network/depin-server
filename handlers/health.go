package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func HandleHealthCheck(c *gin.Context) {
	if os.Getenv("ENABLE_ASSET_UPLOAD") == "true" {
		c.String(http.StatusOK, "OK")
	} else {
		c.String(http.StatusOK, "NOOP")
	}
}
