package cmd

import (
	"flag"
	"fmt"
	"os"
)

func printUsageWithError(err error) {
	fmt.Printf("error: %s", err.Error())
	printUsage()
}

func printUsage() {
	name := "leaderlog-api"
	args := os.Args
	if len(args) > 0 {
		name = args[0]
	}
	fmt.Printf("\nUsage: %s <pool-id> [options]\n", name)
	flag.PrintDefaults()
}

func handleProgramError(err error) {
	if err != nil {
		fmt.Printf("error: %s\n", err.Error())
		os.Exit(1)
	}
}

func handleCLIError(err error) {
	if err != nil {
		printUsageWithError(err)
		os.Exit(1)
	}
}
