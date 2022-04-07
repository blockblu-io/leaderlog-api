package cmd

import (
	"context"
	"flag"
	"fmt"
	"github.com/blockblu-io/leaderlog-api/internal/logging"
	"github.com/blockblu-io/leaderlog-api/pkg/api"
	"github.com/blockblu-io/leaderlog-api/pkg/auth"
	"github.com/blockblu-io/leaderlog-api/pkg/chain/blockfrost"
	"github.com/blockblu-io/leaderlog-api/pkg/chain/syncer"
	"github.com/blockblu-io/leaderlog-api/pkg/db/sqlite"
)

var (
	hostname     string
	port         int
	dbPath       string
	loggingLevel string
)

func Run() {
	flag.StringVar(&hostname, "hostname", "localhost",
		"location at which the API shall be served.")
	flag.IntVar(&port, "port", 9001,
		"port on which the API shall be served.")
	flag.StringVar(&dbPath, "db-path", ".db",
		"path to the directory with the leader log db.")
	flag.StringVar(&loggingLevel, "level", "info",
		"level of logging.")
	flag.Parse()

	poolID := flag.Arg(0)
	if poolID == "" {
		handleCLIError(fmt.Errorf("you must pass the pool ID in hex format as argument"))
	}

	if port <= 0 || port > 65536 {
		handleCLIError(fmt.Errorf("the port must be between 1 and 65536, but was %d",
			port))
	}

	err := logging.InitLogging(loggingLevel)
	handleCLIError(err)

	sqliteDB, err := sqlite.NewSQLiteDB(dbPath)
	handleProgramError(err)
	defer sqliteDB.Close()

	authenticator, err := auth.NewEnvironmentBasedAuthentication()
	handleProgramError(err)

	backend, err := blockfrost.NewBlockFrostBackend()
	handleProgramError(err)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	sync := syncer.NewSyncer(poolID, backend, sqliteDB)
	defer sync.Close()
	go sync.Run(ctx)

	err = api.Serve(hostname, port, sqliteDB, authenticator)
	handleProgramError(err)
}
