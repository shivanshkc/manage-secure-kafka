package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/shivanshkc/msk/internal/config"

	dockerlib "github.com/moby/moby/client"
)

const jvmWarningPrefix = "OpenJDK 64-Bit Server VM warning:"

func CheckHealth(ctx context.Context, configPath, brokerID string) error {
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

	// Check KRaft quorum status.
	if err := checkQuorumStatus(ctx, docker, containerName); err != nil {
		return fmt.Errorf("quorum status check failed: %w", err)
	}

	// Check topics.
	if err := checkTopics(ctx, docker, containerName); err != nil {
		return fmt.Errorf("topic check failed: %w", err)
	}

	// Check SCRAM users.
	if err := checkUsers(ctx, docker, containerName, conf); err != nil {
		return fmt.Errorf("user check failed: %w", err)
	}

	// Check ACLs.
	if err := checkACLs(ctx, docker, containerName); err != nil {
		return fmt.Errorf("ACL check failed: %w", err)
	}

	return nil
}

func checkQuorumStatus(ctx context.Context, docker *dockerlib.Client, containerName string) error {
	cmd := []string{
		"/opt/kafka/bin/kafka-metadata-quorum.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"describe", "--status",
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return err
	}

	fmt.Println("── Quorum ──")
	fmt.Println(cleanOutput(output))
	return nil
}

func checkTopics(ctx context.Context, docker *dockerlib.Client, containerName string) error {
	cmd := []string{
		"/opt/kafka/bin/kafka-topics.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--describe",
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return err
	}

	fmt.Println("\n── Topics ──")
	fmt.Println(cleanOutput(output))
	return nil
}

func checkUsers(ctx context.Context, docker *dockerlib.Client, containerName string, conf config.Config) error {
	cmd := []string{
		"/opt/kafka/bin/kafka-configs.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--describe",
		"--entity-type", "users",
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return err
	}

	// Warn about any configured clients that are missing from the cluster.
	for _, client := range conf.Kafka.Clients {
		if !strings.Contains(output, client.Username) {
			fmt.Printf("  WARNING: configured client %q not found in cluster\n", client.Username)
		}
	}

	fmt.Println("\n── SCRAM Users ──")
	fmt.Println(cleanOutput(output))
	return nil
}

func checkACLs(ctx context.Context, docker *dockerlib.Client, containerName string) error {
	cmd := []string{
		"/opt/kafka/bin/kafka-acls.sh",
		"--bootstrap-server", "localhost:9094",
		"--command-config", commandConfigPath,
		"--list",
	}

	output, err := execInContainer(ctx, docker, containerName, cmd)
	if err != nil {
		return err
	}

	fmt.Println("\n── ACLs ──")
	fmt.Println(cleanOutput(output))
	return nil
}

func cleanOutput(output string) string {
	var lines []string
	for line := range strings.SplitSeq(output, "\n") {
		if strings.HasPrefix(line, jvmWarningPrefix) {
			continue
		}
		lines = append(lines, line)
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
