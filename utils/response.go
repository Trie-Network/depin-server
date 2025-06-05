package utils

import "github.com/gin-gonic/gin"

type JSONResponse struct {
	Status  string      `json:"status"`           // "success" or "error"
	Message string      `json:"message"`          // Human-readable message
	Error   string      `json:"error,omitempty"`  // Optional error detail
	Data    interface{} `json:"data,omitempty"`   // Any payload
}

func RespondSuccess(c *gin.Context, message string, data interface{}) {
	c.JSON(200, JSONResponse{
		Status:  "success",
		Message: message,
		Data:    data,
	})
}

func RespondError(c *gin.Context, code int, message string, err error) {
	resp := JSONResponse{
		Status:  "error",
		Message: message,
	}
	if err != nil {
		resp.Error = err.Error()
	}
	c.JSON(code, resp)
}
