package api

import (
	"github.com/gin-gonic/gin"
	"time"
)

func okPayload(v interface{}) gin.H {
	if v == nil {
		return gin.H{
			"status":    "ok",
			"timestamp": time.Now(),
		}
	}
	return gin.H{
		"status":    "ok",
		"timestamp": time.Now(),
		"response":  v,
	}
}

func errorPayload(message string) gin.H {
	return gin.H{
		"status":    "error",
		"message":   message,
		"timestamp": time.Now(),
	}
}
