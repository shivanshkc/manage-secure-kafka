package handlers

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"

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

// parseBrokerID parses the given string into an integer.
// A broker ID should be between 1 and 9, both inclusive.
func parseBrokerID(id string) (int, error) {
	parsedID, err := strconv.Atoi(id)
	if err != nil {
		return 0, fmt.Errorf("failed to parse broker ID: %w", err)
	}

	if parsedID < 1 || parsedID > 9 {
		return 0, fmt.Errorf("broker ID should be in range [1, 9]")
	}

	return parsedID, nil
}
