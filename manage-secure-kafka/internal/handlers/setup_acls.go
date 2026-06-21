package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/shivanshkc/msk/internal/config"

	dockerlib "github.com/moby/moby/client"
)

func SetupACLs(ctx context.Context, configPath, brokerID string) error {
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

	// Loop over each client's ACLs to grant permissions.
	for _, client := range conf.Kafka.Clients {
		for _, acl := range client.ACLs {
			if slices.Contains(acl.Permissions, "produce") {
				if err := grantProduceACLs(ctx, docker, containerName, client.Username, acl.Topic); err != nil {
					return fmt.Errorf("failed to grant produce ACLs for %s on %s: %w", client.Username, acl.Topic, err)
				}
			}

			if slices.Contains(acl.Permissions, "consume") {
				if err := grantConsumeACLs(ctx, docker, containerName, client.Username, acl.Topic); err != nil {
					return fmt.Errorf("failed to grant consume ACLs for %s on %s: %w", client.Username, acl.Topic, err)
				}
			}
		}
	}

	return nil
}

func grantProduceACLs(ctx context.Context, docker *dockerlib.Client, containerName, username, topic string) error {
	// Write + Describe on the topic.
	cmd := []string{
		"/opt/kafka/bin/kafka-acls.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--add",
		"--allow-principal", "User:" + username,
		"--operation", "Write",
		"--operation", "Describe",
		"--topic", topic,
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return fmt.Errorf("failed to grant topic ACLs: %w", err)
	}
	slog.InfoContext(ctx, "produce ACLs granted on topic", "username", username, "topic", topic, "output", output)

	// Write + Create on transactional ID prefixed with username.
	cmd = []string{
		"/opt/kafka/bin/kafka-acls.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--add",
		"--allow-principal", "User:" + username,
		"--operation", "Write",
		"--operation", "Describe",
		"--transactional-id", username + "-",
		"--resource-pattern-type", "prefixed",
	}

	output, err = execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return fmt.Errorf("failed to grant transactional ID ACLs: %w", err)
	}
	slog.InfoContext(ctx, "produce ACLs granted on transactional ID", "username", username, "output", output)

	return nil
}

func grantConsumeACLs(ctx context.Context, docker *dockerlib.Client, containerName, username, topic string) error {
	// Read + Describe on the topic.
	cmd := []string{
		"/opt/kafka/bin/kafka-acls.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--add",
		"--allow-principal", "User:" + username,
		"--operation", "Read",
		"--operation", "Describe",
		"--topic", topic,
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return fmt.Errorf("failed to grant topic ACLs: %w", err)
	}
	slog.InfoContext(ctx, "consume ACLs granted on topic", "username", username, "topic", topic, "output", output)

	// Read on consumer group prefixed with username.
	cmd = []string{
		"/opt/kafka/bin/kafka-acls.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--add",
		"--allow-principal", "User:" + username,
		"--operation", "Read",
		"--group", username + "-",
		"--resource-pattern-type", "prefixed",
	}

	output, err = execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return fmt.Errorf("failed to grant consumer group ACLs: %w", err)
	}
	slog.InfoContext(ctx, "consume ACLs granted on consumer group", "username", username, "output", output)

	return nil
}
