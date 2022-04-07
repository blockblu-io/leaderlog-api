package api

import (
	"github.com/gin-gonic/gin"
)

// heartbeat returns just an "ok" status object in JSON format. It could be used
// to monitor the reachability of this application.
func heartbeat(router *gin.Engine) {
	router.GET(getPath("heartbeat"), func(c *gin.Context) {
		c.JSON(200, okPayload(nil))
	})
}
