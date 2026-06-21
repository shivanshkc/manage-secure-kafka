package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/shivanshkc/msk/internal/config"

	dockerlib "github.com/moby/moby/client"
)

func CreateTopic(ctx context.Context, configPath, brokerID, topicName string) error {
	if strings.TrimSpace(topicName) == "" {
		return fmt.Errorf("topic name is required")
	}

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

	cmd := []string{
		"/opt/kafka/bin/kafka-topics.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--create",
		"--topic", topicName,
		"--partitions", strconv.Itoa(conf.Kafka.Cluster.NumPartitions),
		"--replication-factor", strconv.Itoa(conf.Kafka.Cluster.ReplicationFactor),
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return fmt.Errorf("failed to create topic %s: %w", topicName, err)
	}

	slog.InfoContext(ctx, "topic created", "topic", topicName, "output", output)
	return nil
}
