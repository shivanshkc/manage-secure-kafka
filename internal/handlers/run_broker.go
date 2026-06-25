package handlers

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/shivanshkc/msk/internal/config"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	dockerlib "github.com/moby/moby/client"
)

const (
	kafkaImageName = "apache/kafka:latest"

	jaasFileContent = `KafkaServer {
		org.apache.kafka.common.security.scram.ScramLoginModule required;
	};`
)

var (
	port9093 = network.MustParsePort("9093")
	port9094 = network.MustParsePort("9094")
	port9095 = network.MustParsePort("9095")
)

func RunBroker(ctx context.Context, configPath, brokerID string) error {
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

	// Special case: If "all" is provided, run all brokers.
	if brokerID == "all" {
		slog.InfoContext(ctx, "running all brokers")
		if err := runAllBrokers(ctx, docker, conf); err != nil {
			return fmt.Errorf("failed to run all brokers: %w", err)
		}
		return nil
	}

	// Parse broker ID to integer.
	parsedID := mustAtoi(brokerID)

	// Run single broker.
	slog.InfoContext(ctx, "running one broker", "id", parsedID)
	if err := runOneBroker(ctx, docker, conf, parsedID); err != nil {
		return fmt.Errorf("failed to run broker %d: %w", parsedID, err)
	}

	return nil
}

func runAllBrokers(ctx context.Context, docker *dockerlib.Client, conf config.Config) error {
	for _, b := range conf.Kafka.Brokers {
		if err := runOneBroker(ctx, docker, conf, b.ID); err != nil {
			return fmt.Errorf("failed to run broker %d: %w", b.ID, err)
		}
	}

	return nil
}

func runOneBroker(ctx context.Context, docker *dockerlib.Client, conf config.Config, brokerID int) error {
	brokerConfig, found := getBrokerConfig(conf, brokerID)
	if !found {
		return fmt.Errorf("failed to find config for broker")
	}

	// Convert to string for various concatenation operations below.
	brokerIDStr := fmt.Sprintf("%d", brokerID)
	brokerName := "broker-" + brokerIDStr
	slog.InfoContext(ctx, "running one broker", "name", brokerName)

	// Remove any old container with confirmation.
	if err := removeWithConfirmation(ctx, docker, brokerName); err != nil {
		return fmt.Errorf("failed to remove existing container: %w", err)
	}

	// Download container image.
	slog.InfoContext(ctx, "pulling the required image")
	if err := pullImage(ctx, docker, kafkaImageName); err != nil {
		return fmt.Errorf("failed to pull kafka image: %w", err)
	}
	slog.InfoContext(ctx, "successfully pulled image")

	// All brokers are super users.
	var superUsers string
	for _, broker := range conf.Kafka.Brokers {
		superUsers += fmt.Sprintf("User:broker-%d;", broker.ID)
	}
	// Remove trailing semicolon.
	superUsers = strings.TrimSuffix(superUsers, ";")

	// List of brokers that will participate in the KRaft consensus protocol.
	// It is in the format: `1@addr,2@addr,3@addr`
	var controllerQuorumVoters string
	for _, broker := range conf.Kafka.Brokers {
		controllerQuorumVoters += fmt.Sprintf(`%d@%s:%d9093,`, broker.ID, broker.InternalHost, broker.ID)
	}
	// Remove trailing comma.
	controllerQuorumVoters = strings.TrimSuffix(controllerQuorumVoters, ",")

	// The advertised listeners list.
	// The ports here need to be the published ones (19093, and not 9093)
	advertisedListeners := fmt.Sprintf(`INTERNAL://%s:%s9094,EXTERNAL://%s:%s9095`,
		brokerConfig.InternalHost, brokerIDStr, brokerConfig.ExternalHost, brokerIDStr)

	// Create directory for JAAS file.
	jaasFileParentPath := filepath.Join(brokerConfig.VolumePath, "conf")
	if err := os.MkdirAll(jaasFileParentPath, 0755); err != nil {
		return fmt.Errorf("failed to create conf directory inside volume path: %w", err)
	}

	// Remove old jaas file.
	jaasFilePath := filepath.Join(jaasFileParentPath, "jaas.conf")
	if err := os.Remove(jaasFilePath); err != nil {
		// Ignore not-exists error.
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("failed to remove old jaas file: %w", err)
		}
	}

	// Create new jaas file.
	if err := os.WriteFile(jaasFilePath, []byte(jaasFileContent), 0440); err != nil {
		return fmt.Errorf("failed to create jaas file for broker: %w", err)
	}

	containerCreateOptions := dockerlib.ContainerCreateOptions{
		Name: brokerName,
		HostConfig: &container.HostConfig{
			PortBindings: network.PortMap{
				port9093: []network.PortBinding{{HostPort: brokerIDStr + "9093"}},
				port9094: []network.PortBinding{{HostPort: brokerIDStr + "9094"}},
				port9095: []network.PortBinding{{HostPort: brokerIDStr + "9095"}},
			},
			RestartPolicy: container.RestartPolicy{Name: container.RestartPolicyUnlessStopped},
			Binds: getBrokerVolumeBinds(
				brokerConfig.TLS.KeystorePath,
				brokerConfig.VolumePath,
				conf.Kafka.TLS.TruststorePath,
			),
		},
		Config: &container.Config{
			Image: kafkaImageName,
			ExposedPorts: network.PortSet{
				port9093: struct{}{},
				port9094: struct{}{},
				port9095: struct{}{},
			},
			Env: []string{
				"CLUSTER_ID=" + conf.Kafka.Cluster.ID,
				"KAFKA_OPTS=-Djava.security.auth.login.config=/etc/kafka/secrets/jaas.conf",
				"KAFKA_NODE_ID=" + brokerIDStr,
				"KAFKA_PROCESS_ROLES=broker,controller",
				"KAFKA_CONTROLLER_QUORUM_VOTERS=" + controllerQuorumVoters,
				// Listener config.
				"KAFKA_LISTENERS=CONTROLLER://:9093,INTERNAL://:9094,EXTERNAL://:9095",
				"KAFKA_ADVERTISED_LISTENERS=" + advertisedListeners,
				"KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=CONTROLLER:SSL,INTERNAL:SSL,EXTERNAL:SASL_SSL",
				"KAFKA_CONTROLLER_LISTENER_NAMES=CONTROLLER",
				"KAFKA_INTER_BROKER_LISTENER_NAME=INTERNAL",
				// TLS config.
				"KAFKA_SSL_KEYSTORE_LOCATION=/etc/kafka/secrets/broker.keystore.p12",
				"KAFKA_SSL_KEYSTORE_PASSWORD=" + brokerConfig.TLS.KeystorePassword,
				"KAFKA_SSL_KEY_PASSWORD=" + brokerConfig.TLS.KeyPassword,
				"KAFKA_SSL_TRUSTSTORE_LOCATION=/etc/kafka/secrets/broker.truststore.p12",
				"KAFKA_SSL_TRUSTSTORE_PASSWORD=" + conf.Kafka.TLS.TruststorePassword,
				"KAFKA_SSL_KEYSTORE_TYPE=PKCS12",
				"KAFKA_SSL_TRUSTSTORE_TYPE=PKCS12",
				// mTLS for inter-broker comms.
				"KAFKA_LISTENER_NAME_CONTROLLER_SSL_CLIENT_AUTH=required",
				"KAFKA_LISTENER_NAME_INTERNAL_SSL_CLIENT_AUTH=required",
				// No mTLS for broker-client comms.
				"KAFKA_LISTENER_NAME_EXTERNAL_SSL_CLIENT_AUTH=none",
				// SASL
				"KAFKA_SASL_ENABLED_MECHANISMS=SCRAM-SHA-512",
				"KAFKA_LISTENER_NAME_EXTERNAL_SASL_ENABLED_MECHANISMS=SCRAM-SHA-512",
				// Authorization
				"KAFKA_AUTHORIZER_CLASS_NAME=org.apache.kafka.metadata.authorizer.StandardAuthorizer",
				"KAFKA_SSL_PRINCIPAL_MAPPING_RULES=RULE:^CN=(.*?),OU=.*$/$1/,DEFAULT",
				"KAFKA_SUPER_USERS=" + superUsers,
				"KAFKA_ALLOW_EVERYONE_IF_NO_ACL_FOUND=false",
				// Storage and replication.
				"KAFKA_LOG_DIRS=/var/lib/kafka/data",
				"KAFKA_NUM_PARTITIONS=" + strconv.Itoa(conf.Kafka.Cluster.NumPartitions),
				"KAFKA_DEFAULT_REPLICATION_FACTOR=" + strconv.Itoa(conf.Kafka.Cluster.ReplicationFactor),
				"KAFKA_MIN_INSYNC_REPLICAS=" + strconv.Itoa(conf.Kafka.Cluster.MinInsyncReplicas),
				"KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=" + strconv.Itoa(conf.Kafka.Cluster.ReplicationFactor),
			},
		},
	}

	// Create container.
	result, err := docker.ContainerCreate(ctx, containerCreateOptions)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if len(result.Warnings) == 0 {
		slog.InfoContext(ctx, "container created without any warnings")
	}

	// Log warnings if any.
	for _, warning := range result.Warnings {
		slog.WarnContext(ctx, "container created with warnings", "message", warning)
	}

	// Start container.
	if _, err := docker.ContainerStart(ctx, result.ID, dockerlib.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	slog.InfoContext(ctx, "container is running", "id", result.ID)
	return nil
}

func removeWithConfirmation(ctx context.Context, docker *dockerlib.Client, brokerName string) error {
	result, err := docker.ContainerList(ctx, dockerlib.ContainerListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var alreadyExists bool
	for _, item := range result.Items {
		if slices.Contains(item.Names, "/"+brokerName) {
			alreadyExists = true
			break
		}
	}

	if !alreadyExists {
		slog.DebugContext(ctx, "no existing containers found")
		return nil
	}

	fmt.Printf("A container with name %s already exists. Do you want to remove it? (y/n): ", brokerName)
	var answer string
	_, _ = fmt.Scanf("%s", &answer)

	switch strings.ToLower(answer) {
	case "y", "yes":
		if _, err := docker.ContainerStop(ctx, brokerName, dockerlib.ContainerStopOptions{}); err != nil {
			return fmt.Errorf("failed to stop the old container: %w", err)
		}
		slog.DebugContext(ctx, "container successfully stopped", "name", brokerName)

		if _, err := docker.ContainerRemove(ctx, brokerName, dockerlib.ContainerRemoveOptions{}); err != nil {
			return fmt.Errorf("failed to remove the old container: %w", err)
		}
		slog.InfoContext(ctx, "container successfully removed", "name", brokerName)

		return nil
	case "n", "no":
		return fmt.Errorf("user canceled operation")
	default:
		return fmt.Errorf("user provided unknown input")
	}
}
