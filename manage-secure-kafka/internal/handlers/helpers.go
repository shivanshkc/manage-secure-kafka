package handlers

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/docker/docker/pkg/stdcopy"
	dockerlib "github.com/moby/moby/client"
	"github.com/shivanshkc/msk/internal/config"
)

// getBrokerVolumeBinds returns the volume binds for a Kafka broker.
// These can be directly used with HostConfig.Binds in docker's ContainerCreateOptions.
func getBrokerVolumeBinds(brokerKeystorePath, brokerVolumePath, truststorePath string) []string {
	return []string{
		brokerKeystorePath + ":/etc/kafka/secrets/broker.keystore.p12:ro",
		truststorePath + ":/etc/kafka/secrets/broker.truststore.p12:ro",
		filepath.Join(brokerVolumePath, "conf/jaas.conf") + ":/etc/kafka/secrets/jaas.conf:ro",
		filepath.Join(brokerVolumePath, "data") + ":/var/lib/kafka/data",
	}
}

// getBrokerConfig returns the broker's config whose ID matches the given ID.
func getBrokerConfig(conf config.Config, brokerID int) (config.KafkaBrokerConfig, bool) {
	for _, c := range conf.Kafka.Brokers {
		if c.ID == brokerID {
			return c, true
		}
	}

	return config.KafkaBrokerConfig{}, false
}

// pullImage pulls the given container image using the given docker client.
// It returns when the image pull is complete or if any error occurs.
func pullImage(ctx context.Context, docker *dockerlib.Client, image string) error {
	// Initiate image pull.
	imagePullResponse, err := docker.ImagePull(ctx, image, dockerlib.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to initiate image pull operation: %w", err)
	}

	// Wait for completion.
	if err := imagePullResponse.Wait(ctx); err != nil {
		return fmt.Errorf("failed to complete image pull operation: %w", err)
	}

	return nil
}

// execInContainer executes the given command in the given container.
func execInContainer(ctx context.Context, docker *dockerlib.Client, containerID string, cmd []string) (string, error) {
	execResult, err := docker.ExecCreate(ctx, containerID, dockerlib.ExecCreateOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create exec: %w", err)
	}

	attachResult, err := docker.ExecAttach(ctx, execResult.ID, dockerlib.ExecAttachOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to attach exec: %w", err)
	}
	defer attachResult.Close()

	var buf bytes.Buffer
	stdcopy.StdCopy(&buf, &buf, attachResult.Reader)

	inspectResult, err := docker.ExecInspect(ctx, execResult.ID, dockerlib.ExecInspectOptions{})
	if err != nil {
		return buf.String(), fmt.Errorf("failed to inspect exec: %w", err)
	}

	if inspectResult.ExitCode != 0 {
		return buf.String(), fmt.Errorf("command exited with code %d: %s", inspectResult.ExitCode, buf.String())
	}

	return buf.String(), nil
}

// mustAtoi converts the given string into int. It assumes that the string is a valid integer.
func mustAtoi(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
