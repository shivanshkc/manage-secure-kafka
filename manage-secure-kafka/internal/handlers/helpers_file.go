package handlers

import (
	"context"
	"fmt"

	"github.com/shivanshkc/msk/internal/config"

	dockerlib "github.com/moby/moby/client"
)

const commandConfigPath = "/tmp/command.properties"

// writeCommandConfig writes a file that contains SSL config into the given broker container.
func writeCommandConfig(ctx context.Context, docker *dockerlib.Client, containerName string,
	brokerConfig config.KafkaBrokerConfig, conf config.Config,
) error {
	content := fmt.Sprintf(`security.protocol=SSL
ssl.keystore.location=/etc/kafka/secrets/broker.keystore.p12
ssl.keystore.password=%s
ssl.key.password=%s
ssl.truststore.location=/etc/kafka/secrets/broker.truststore.p12
ssl.truststore.password=%s
ssl.keystore.type=PKCS12
ssl.truststore.type=PKCS12
`,
		brokerConfig.TLS.KeystorePassword,
		brokerConfig.TLS.KeyPassword,
		conf.Kafka.TLS.TruststorePassword,
	)

	return writeFileInContainer(ctx, docker, containerName, commandConfigPath, content)
}

// writeFileInContainer writes a file at the given path, with given content, in the given container.
func writeFileInContainer(ctx context.Context, docker *dockerlib.Client, containerID, path, content string) error {
	cmd := []string{"sh", "-c", fmt.Sprintf("cat > %s << 'CMDEOF'\n%s\nCMDEOF", path, content)}
	_, err := execInContainer(ctx, docker, containerID, cmd)
	return err
}
