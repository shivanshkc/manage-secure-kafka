package handlers

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/shivanshkc/msk/internal/config"

	dockerlib "github.com/moby/moby/client"
)

func SetupUsers(ctx context.Context, configPath, brokerID string) error {
	// All config.
	conf, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	slog.DebugContext(ctx, "successfully loaded config")

	// Docker client.
	docker, err := dockerlib.New()
	if err != nil {
		return fmt.Errorf("failed to create docker client: %w", err)
	}
	slog.DebugContext(ctx, "successfully created the docker client")

	// Get the required broker's config.
	brokerConfig, found := getBrokerConfig(conf, mustAtoi(brokerID))
	if !found {
		return fmt.Errorf("broker %s not found in config", brokerID)
	}

	containerName := "broker-" + brokerID
	// Command config is required because of mTLS.
	if err := writeCommandConfig(ctx, docker, containerName, brokerConfig, conf); err != nil {
		return fmt.Errorf("failed to write command config: %w", err)
	}

	// Loop over each user to create them.
	for _, client := range conf.Kafka.Clients {
		// Form the command to create user.
		cmd := []string{
			"/opt/kafka/bin/kafka-configs.sh",
			"--bootstrap-server", "localhost:9094",
			"--command-config", commandConfigPath,
			"--alter",
			"--add-config", fmt.Sprintf("SCRAM-SHA-512=[password=%s]", client.Password),
			"--entity-type", "users",
			"--entity-name", client.Username,
		}

		// Execute command in the container.
		output, err := execInContainer(ctx, docker, containerName, cmd)
		if err != nil {
			return fmt.Errorf("failed to create SCRAM user %s: %w", client.Username, err)
		}

		slog.InfoContext(ctx, "SCRAM user created", "username", client.Username, "output", output)
	}

	return nil
}
