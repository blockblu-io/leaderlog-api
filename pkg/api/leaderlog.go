package api

import (
	"fmt"
	"github.com/blockblu-io/leaderlog-api/pkg/api/dto"
	"github.com/blockblu-io/leaderlog-api/pkg/auth"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

// groupByDates takes a look at the given leader log and groups the assigned
// blocks by the day for which they are planned. The grouping is based on the
// given location (i.e. dependent on timezone).
func groupByDates(log *db.LeaderLog, loc *time.Location) map[time.Time][]uint {
	groupedDates := make(map[time.Time][]uint)
	for _, block := range log.Blocks {
		t := block.Timestamp.In(loc)
		key := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
		list, found := groupedDates[key]
		if !found {
			list = make([]uint, 0)
		}
		list = append(list, block.No)
		groupedDates[key] = list
	}
	return groupedDates
}

// groupByStatus takes a look at the given leader log and groups the assigned
// blocks by their status.
func groupByStatus(log *db.LeaderLog) map[db.BlockStatus]uint {
	groupedStatus := make(map[db.BlockStatus]uint)
	for i := db.NotMinted; i < db.GHOSTED; i++ {
		groupedStatus[i] = 0
	}
	for _, block := range log.Blocks {
		count, found := groupedStatus[block.Status]
		if !found {
			continue
		}
		groupedStatus[block.Status] = count + 1
	}
	return groupedStatus
}

func computeMaxPerformance(expectedBlockNumber float32, statusMap map[db.BlockStatus]uint) float64 {
	return float64(statusMap[db.Minted]+statusMap[db.NotMinted]) / float64(expectedBlockNumber)
}

func handleLeaderLogFetching(db db.DB, c *gin.Context) (*db.LeaderLog, error) {
	epoch, err := strconv.Atoi(c.Param("epoch"))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, errorPayload("epoch couldn't be parsed"))
		return nil, err
	}
	log, err := db.GetLeaderLog(c, uint(epoch))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, errorPayload(err.Error()))
		return nil, err
	}
	if log == nil {
		c.AbortWithStatusJSON(http.StatusNotFound, errorPayload("no log for epoch could be found"))
		return nil, fmt.Errorf("couldn't find a log for this epoch")
	}
	return log, nil
}

func getRegisteredEpochs(idb db.DB) func(router *gin.Engine) {
	return func(router *gin.Engine) {
		router.GET(getPath("epoch"), func(c *gin.Context) {
			var limit uint = 10
			limitParam := c.Query("limit")
			if limitParam != "" {
				limitVal, err := strconv.Atoi(limitParam)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest,
						errorPayload("the given limit query parameter couldn't be parsed"))
					return
				}
				limit = uint(limitVal)
			}
			epochs, err := idb.GetRegisteredEpochs(c, db.OrderingDesc, limit)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, errorPayload(err.Error()))
				return
			}
			c.JSON(200, okPayload(epochs))
		})
	}
}

func getLeaderLogPerformance(idb db.DB) func(router *gin.Engine) {
	return func(router *gin.Engine) {
		router.GET(getPath("epoch/:epoch/performance"), func(c *gin.Context) {
			log, err := handleLeaderLogFetching(idb, c)
			if err != nil {
				return
			}
			groupedMap := groupByStatus(log)
			assignedBlock := len(log.Blocks)
			c.JSON(200, okPayload(gin.H{
				"epoch":               log.Epoch,
				"assignedBlocks":      assignedBlock,
				"expectedBlockNumber": log.ExpectedBlockNumber,
				"maxPerformance":      computeMaxPerformance(log.ExpectedBlockNumber, groupedMap),
				"status": gin.H{
					"notMinted":      groupedMap[db.NotMinted],
					"minted":         groupedMap[db.Minted],
					"doubleAssigned": groupedMap[db.DoubleAssignment],
					"heightBattle":   groupedMap[db.HeightBattle],
				},
			}))
		})
	}
}

func getLeaderLogByDate(db db.DB) func(router *gin.Engine) {
	return func(router *gin.Engine) {
		router.GET(getPath("epoch/:epoch/by/date/:location"), func(c *gin.Context) {
			loc, err := time.LoadLocation(c.Param("location"))
			if err != nil {
				c.JSON(400, errorPayload(err.Error()))
				return
			}
			log, err := handleLeaderLogFetching(db, c)
			if err != nil {
				return
			}
			c.JSON(200, okPayload(groupByDates(log, loc)))
		})
	}
}

func postLeaderLog(db db.DB, auth auth.Authenticator) func(router *gin.Engine) {
	return func(router *gin.Engine) {
		router.POST(getPath(""), func(c *gin.Context) {
			username, password, ok := c.Request.BasicAuth()
			if !ok || !auth.CheckAuthentication(username, password) {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					errorPayload("you aren't authorized to call this method"))
				return
			}
			reader := c.Request.Body
			defer reader.Close()
			log, err := dto.ParseLeaderLog(reader)
			if err != nil {
				if err == dto.ParsingError {
					c.AbortWithStatusJSON(http.StatusBadRequest, errorPayload(err.Error()))
				} else {
					c.AbortWithStatusJSON(http.StatusInternalServerError, errorPayload(err.Error()))
				}
				return
			}
			err = db.WriteLeaderLog(c, log.ToPlain())
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, errorPayload(err.Error()))
			} else {
				c.JSON(200, okPayload(nil))
			}
		})
	}
}
