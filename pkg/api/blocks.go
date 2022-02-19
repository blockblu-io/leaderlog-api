package api

import (
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func getAssignedBlocksBeforeNow(idb db.DB) func(router *gin.Engine) {
	return func(router *gin.Engine) {
		router.GET(getPath("epoch/:epoch/blocks/before/now"), func(c *gin.Context) {
			epoch, err := strconv.Atoi(c.Param("epoch"))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, errorPayload("epoch couldn't be parsed"))
				return
			}
			blocks, err := idb.GetAssignedBlocksBeforeNow(c, uint(epoch))
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, errorPayload(err.Error()))
				return
			}
			c.JSON(200, okPayload(blocks))
		})
	}
}
