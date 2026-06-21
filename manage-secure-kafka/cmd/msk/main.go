package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strconv"
	"strings"
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

	// Add flgs.
	configPath := flag.String("config", "", "Path to config file")
	brokerID := flag.String("broker", "",
		`For run-broker: ID of the broker to run. Provide "all" to run all brokers.
For setup-users and setup-acls: ID of the broker to use for the operation.
For check-health: ID of the broker to check. Provide "all" to check all brokers.`)

	// Here flag.Parse does not work because it parses from os.Args[1:]
	// os.Args[1] is the command itself, which is not a flag argument so the parser stops before
	// ever reaching the actual flags.
	flag.CommandLine.Parse(os.Args[2:])

	// Validate flag inputs.
	if err := validateArgs(*configPath, *brokerID); err != nil {
		panic("invalid inputs: " + err.Error())
	}

	// Run the command.
	if err := runCommand(ctx, os.Args[1], *configPath, *brokerID); err != nil {
		panic("failed to execute command: " + err.Error())
	}
}

// runCommand runs the given command with the given arguments.
func runCommand(ctx context.Context, command, configPath, brokerID string) error {
	switch command {
	case "run-broker":
		return handlers.RunBroker(ctx, configPath, brokerID)
	case "setup-users":
		return handlers.SetupUsers(ctx, configPath, brokerID)
	case "setup-acls":
		return handlers.SetupACLs(ctx, configPath, brokerID)
	case "check-health":
		return handlers.CheckHealth(ctx, configPath, brokerID)
	default:
		return fmt.Errorf("unknown command provided: %s", command)
	}
}

// validateArgs validates all the given args and returns the first error.
func validateArgs(configPath, brokerID string) error {
	if err := validateConfigPath(configPath); err != nil {
		return err
	}

	if err := validateBrokerID(brokerID); err != nil {
		return err
	}

	return nil
}

// validateConfigPath checks if the given path exists and points to a file.
func validateConfigPath(configPath string) error {
	if strings.TrimSpace(configPath) == "" {
		return fmt.Errorf("config path is required")
	}

	info, err := os.Stat(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("provided file does not exist")
		}
		return fmt.Errorf("failed to check file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("provided path is a directory, not a file")
	}

	return nil
}

// validateBrokerID checks if the given broker ID is either "all" or an integer in [1, 9].
func validateBrokerID(brokerID string) error {
	if strings.TrimSpace(brokerID) == "" {
		return fmt.Errorf("broker is required")
	}

	if brokerID == "all" {
		return nil
	}

	parsedID, err := strconv.Atoi(brokerID)
	if err != nil {
		return fmt.Errorf(`broker should either be "all" or an integer in [1, 9]`)
	}

	if parsedID < 1 || parsedID > 9 {
		return fmt.Errorf(`broker should either be "all" or an integer in [1, 9]`)
	}

	return nil
}
