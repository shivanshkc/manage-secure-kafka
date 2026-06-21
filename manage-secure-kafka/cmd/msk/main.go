package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/shivanshkc/msk/internal/handlers"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// A command is required.
	if len(os.Args) < 2 {
		panic("no command provided")
	}

	// Many command require just the config flag. So, avoid redeclaration.
	commonFlagSet := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	configPath := commonFlagSet.String("config", "", "Path to config file")

	switch os.Args[1] {
	case "run-broker":
		// Add other required flags.
		brokerID := commonFlagSet.String("broker", "",
			"ID of the broker to run. Provide 'all' to run all brokers")

		// Parse all flags.
		if err := commonFlagSet.Parse(os.Args[2:]); err != nil {
			panic("failed to parse flags: " + err.Error())
		}

		// Execute command.
		if err := handlers.RunBroker(ctx, *configPath, *brokerID); err != nil {
			panic("error in command execution: " + err.Error())
		}
	case "setup-users":
		if err := commonFlagSet.Parse(os.Args[2:]); err != nil {
			panic("failed to parse flags: " + err.Error())
		}

		// Execute command.
		if err := handlers.SetupUsers(ctx, *configPath); err != nil {
			panic("error in command execution: " + err.Error())
		}
	case "setup-acls":
		if err := commonFlagSet.Parse(os.Args[2:]); err != nil {
			panic("failed to parse flags: " + err.Error())
		}

		// Execute command.
		if err := handlers.SetupACLs(ctx, *configPath); err != nil {
			panic("error in command execution: " + err.Error())
		}
	case "check-health":
		if err := commonFlagSet.Parse(os.Args[2:]); err != nil {
			panic("failed to parse flags: " + err.Error())
		}

		// Execute command.
		if err := handlers.CheckHealth(ctx, *configPath); err != nil {
			panic("error in command execution: " + err.Error())
		}
	default:
		panic("unknown command provided: " + os.Args[1])
	}
}
