package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type KafkaConfig struct {
	Cluster struct {
		ID                string `json:"id"`
		NumPartitions     int    `json:"numPartitions"`
		ReplicationFactor int    `json:"replicationFactor"`
		MinInsyncReplicas int    `json:"minInsyncReplicas"`
	} `json:"cluster"`

	TLS struct {
		TruststorePath     string `json:"truststorePath"`
		TruststorePassword string `json:"truststorePassword"`
	} `json:"tls"`

	Brokers []KafkaBrokerConfig `json:"brokers"`

	Clients []struct {
		Username string `json:"username"`
		Password string `json:"password"`
		ACLs     []struct {
			Topic       string
			Permissions []string
		} `json:"acls"`
	} `json:"clients"`
}

type KafkaBrokerConfig struct {
	ID           int    `json:"id"`
	InternalHost string `json:"internalHost"`
	ExternalHost string `json:"externalHost"`
	VolumePath   string `json:"volumePath"`
	TLS          struct {
		KeystorePath     string `json:"keystorePath"`
		KeystorePassword string `json:"keystorePassword"`
		KeyPassword      string `json:"keyPassword"`
	} `json:"tls"`
}

// Config encapsulates all config required by the application.
type Config struct {
	Kafka       KafkaConfig `json:"kafkaConfig"`
	LoggerLevel string      `json:"loggerLevel"`
}

// Load config from the given JSON file.
func Load(jsonPath string) (Config, error) {
	content, err := os.ReadFile(jsonPath)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config file at %s because: %w", jsonPath, err)
	}

	var config Config
	if err := json.Unmarshal(content, &config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config file at %s because: %w", jsonPath, err)
	}

	if err := validate(config); err != nil {
		return Config{}, fmt.Errorf("config is invalid: %w", err)
	}

	return config, nil
}

// validate the loaded config.
func validate(conf Config) error {
	if conf.LoggerLevel == "" {
		return fmt.Errorf("loggerLevel is required")
	}

	return nil
}
