# manage-secure-kafka (msk)

CLI tool to deploy and manage a secure Kafka cluster using Docker. Configures mTLS for inter-broker communication and SCRAM-SHA-512 for client authentication. Uses KRaft (no ZooKeeper).

## Prerequisites

Before using this tool, generate TLS material (CA, keystores, truststore) using the scripts in `scripts/`. The tool assumes these files already exist at the paths specified in the config.

```bash
# Generate certs for all brokers (run once)
./scripts/0-all.sh
```

Each broker needs its own keystore. All brokers share one truststore (containing the CA cert). Ensure the certificate SANs include the hostnames used in your config (e.g., `host.docker.internal` for local deployments).

## Commands

All commands require `-config` and `-broker` flags.

```
msk <command> -config <path> -broker <all|id> [flags]
```

### run-broker

Deploy one or more Kafka broker containers.

```bash
msk run-broker -config config.json -broker all   # local: deploy all brokers
msk run-broker -config config.json -broker 1     # prod: deploy broker 1
```

Idempotent: prompts to remove an existing container with the same name before creating a new one.

### create-topic

Create a topic. Partition count and replication factor come from the cluster config.

```bash
msk create-topic -config config.json -broker 1 -topic orders
```

### setup-users

Create SCRAM-SHA-512 credentials for each client defined in the config.

```bash
msk setup-users -config config.json -broker 1
```

Idempotent: creating a user that already exists updates it.

### setup-acls

Grant ACL permissions for each client based on their `acls` config.

```bash
msk setup-acls -config config.json -broker 1
```

Idempotent: adding an existing ACL is a no-op.

### check-health

Check cluster health: quorum status, topics, SCRAM users, and ACLs.

```bash
msk check-health -config config.json -broker 1
```

Warns if any client defined in the config is missing from the cluster.

## Typical workflow

```bash
# 1. Generate TLS material
./scripts/0-all.sh

# 2. Deploy brokers
msk run-broker -config config.json -broker all

# 3. Create topics
msk create-topic -config config.json -broker 1 -topic orders
msk create-topic -config config.json -broker 1 -topic events

# 4. Create SCRAM users
msk setup-users -config config.json -broker 1

# 5. Set up ACLs
msk setup-acls -config config.json -broker 1

# 6. Verify
msk check-health -config config.json -broker 1
```

## Config

```json
{
  "loggerLevel": "info",
  "kafkaConfig": {
    "cluster": {
      "id": "lVTckYCITwWSX_siiDBoaw",
      "numPartitions": 3,
      "replicationFactor": 3,
      "minInsyncReplicas": 2
    },
    "tls": {
      "truststorePath": "$HOME/.secure-kafka/tls/broker.truststore.p12",
      "truststorePassword": "secret-pass"
    },
    "brokers": [
      {
        "id": 1,
        "internalHost": "host.docker.internal",
        "externalHost": "localhost",
        "volumePath": "$HOME/.secure-kafka/volumes/broker-1",
        "tls": {
          "keystorePath": "$HOME/.secure-kafka/tls/broker-1.keystore.p12",
          "keystorePassword": "secret-pass",
          "keyPassword": "secret-pass"
        }
      }
    ],
    "clients": [
      {
        "username": "service-a",
        "password": "service-a-secret",
        "acls": [
          { "topic": "orders", "permissions": ["produce", "consume"] },
          { "topic": "events", "permissions": ["produce"] }
        ]
      }
    ]
  }
}
```

### Cluster config

| Field | Description |
|---|---|
| `cluster.id` | KRaft cluster ID. Generate once with `kafka-storage.sh random-uuid`. Same on every node. |
| `cluster.numPartitions` | Default partition count for new topics. |
| `cluster.replicationFactor` | Default replication factor for new topics. |
| `cluster.minInsyncReplicas` | Minimum replicas that must ack a write when producer uses `acks=all`. |
| `tls.truststorePath` | Absolute path to the shared truststore (contains the CA cert). |
| `tls.truststorePassword` | Truststore password. |

### Per-broker config

| Field | Description |
|---|---|
| `id` | Unique node ID (1-9). |
| `internalHost` | Address for inter-broker and controller traffic. Use `host.docker.internal` for local deployments. |
| `externalHost` | Address for client traffic. Use `localhost` for local deployments. |
| `volumePath` | Host path for broker data and config files. |
| `tls.keystorePath` | Absolute path to this broker's keystore. |
| `tls.keystorePassword` | Keystore password. |
| `tls.keyPassword` | Private key password. |

### Client config

| Field | Description |
|---|---|
| `username` | SCRAM-SHA-512 username. |
| `password` | SCRAM-SHA-512 password. |
| `acls` | Array of `{topic, permissions}`. Permissions: `"produce"`, `"consume"`. |

| Permission | Kafka operations granted |
|---|---|
| `produce` | Write, Describe on topic. Write, Describe on transactional ID (username prefix). |
| `consume` | Read, Describe on topic. Read on consumer group (username prefix). |

## Listeners

| Listener | Container Port | Protocol | Auth | Purpose |
|---|---|---|---|---|
| CONTROLLER | 9093 | SSL | mTLS | KRaft consensus |
| INTERNAL | 9094 | SSL | mTLS | Inter-broker replication |
| EXTERNAL | 9095 | SASL_SSL | SCRAM-SHA-512 | Client connections |

Host ports are derived as `{brokerID}909{3,4,5}` (e.g., broker 1 maps to 19093/19094/19095).

## Derived by the tool

- `controller.quorum.voters` from the brokers list using `internalHost`.
- `super.users` as `User:broker-{id}` for every broker.
- `ssl.principal.mapping.rules` to extract CN from certificate DN.
- Container name as `broker-{id}`.
- JAAS config for SCRAM, generated and mounted automatically.
- Docker port mappings from broker ID to container-internal ports.
