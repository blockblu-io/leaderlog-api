package api

import (
	"fmt"
	"github.com/blockblu-io/leaderlog-api/pkg/auth"
	"github.com/blockblu-io/leaderlog-api/pkg/db"
	"github.com/blockblu-io/leaderlog-api/pkg/logging"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const RootPath = "leaderlog"

// getPath assembles the path for api calls given the relative path.
// This functions returns the complete path that can be passed to the
// gin framework.
func getPath(relativePath string) string {
	return fmt.Sprintf("%s/%s", RootPath, relativePath)
}

// routes returns a list of all routes for this api.
func routes(db db.DB, auth auth.Authenticator) []func(router *gin.Engine) {
	return []func(*gin.Engine){
		heartbeat,
		getRegisteredEpochs(db),
		postLeaderLog(db, auth),
		getLeaderLogByDate(db),
		getLeaderLogPerformance(db),
		getAssignedBlocksBeforeNow(db),
	}
}

// Serve starts the API at the given hostname and on the given port.
func Serve(hostname string, port int, db db.DB, auth auth.Authenticator) error {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(logging.GinLoggingHook(), gin.Recovery())
	_ = router.SetTrustedProxies(nil)
	for _, function := range routes(db, auth) {
		function(router)
	}
	address := fmt.Sprintf("%s:%d", hostname, port)
	log.Infof("starting the API at address '%s'", address)
	return router.Run(address)
}
