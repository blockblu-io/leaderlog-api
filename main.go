package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/blockblu-io/leaderlog-api/pkg/api"
	"github.com/blockblu-io/leaderlog-api/pkg/auth"
	"github.com/blockblu-io/leaderlog-api/pkg/chain/blockfrost"
	"github.com/blockblu-io/leaderlog-api/pkg/chain/syncer"
	"github.com/blockblu-io/leaderlog-api/pkg/db/sqlite"
	"github.com/blockblu-io/leaderlog-api/pkg/logging"
	"os"
)

var (
	hostname     string
	port         int
	dbPath       string
	loggingLevel string
)

func printHelp() {
	name := "leaderlog-api"
	args := os.Args
	if len(args) > 0 {
		name = args[0]
	}
	fmt.Printf("\nUsage: %s <pool-id> [options]\n", name)
	flag.PrintDefaults()
}

func handleError(err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}

func main() {
	flag.StringVar(&hostname, "hostname", "localhost", "location at which the API shall be served.")
	flag.IntVar(&port, "port", 9001, "port on which the API shall be served.")
	flag.StringVar(&dbPath, "db-path", ".db", "path to the directory with the leader log db.")
	flag.StringVar(&loggingLevel, "level", "info", "level of logging.")
	flag.Parse()

	poolID := flag.Arg(0)
	if poolID == "" {
		fmt.Print("error: you must pass the pool ID in hex format as argument\n")
		printHelp()
		os.Exit(1)
	}

	if port <= 0 || port > 65536 {
		fmt.Printf("error: the port must be between 1 and 65536, but was %d\n", port)
		printHelp()
		os.Exit(1)
	}

	err := logging.InitLogging(loggingLevel)
	handleError(err)

	sqliteDB, err := sqlite.NewSQLiteDB(dbPath)
	handleError(err)
	defer sqliteDB.Close()

	authenticator, err := auth.NewEnvironmentBasedAuthentication()
	handleError(err)

	backend, err := blockfrost.NewBlockFrostBackend()
	handleError(err)

	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	sync := syncer.NewSyncer(poolID, backend, sqliteDB)
	defer sync.Close()
	go sync.Run(ctx)

	err = api.Serve(hostname, port, sqliteDB, authenticator)
	handleError(err)
}
