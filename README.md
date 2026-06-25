# msk

Deploy and manage a secure Apache Kafka cluster using Docker.

- mTLS for inter-broker communication
- SCRAM-SHA-512 for client authentication
- KRaft mode (no ZooKeeper)
- Works with the `apache/kafka` Docker image

## Quick start (local)

```bash
# 1. Generate TLS certificates
./scripts/0-all.sh

# 2. Copy the TLS output to each broker machine (skip for local)
#    Files are at: ~/.secure-kafka/tls/

# 3. Deploy all 3 brokers locally
msk run-broker -config config.json -broker all

# 4. Create a topic
msk create-topic -config config.json -broker 1 -topic my-topic

# 5. Create SCRAM users and ACLs
msk setup-users -config config.json -broker 1
msk setup-acls  -config config.json -broker 1

# 6. Verify everything is healthy
msk check-health -config config.json -broker 1
```

## TLS scripts

The `scripts/` directory contains shell scripts that generate all TLS material needed by the cluster: a CA, per-broker keystores, and a shared truststore.

### Running the scripts

```bash
./scripts/0-all.sh
```

This script:
1. Stops and removes any running broker containers (`broker-1`, `broker-2`, `broker-3`)
2. Deletes existing volumes at `~/.secure-kafka/volumes/`
3. Deletes existing TLS data at `~/.secure-kafka/tls/`
4. Generates fresh CA, keystores, signed certs, and truststore for all 3 brokers

For prod deployments where brokers run on remote machines, step 1 is a no-op (no local containers to remove).

### Output

All TLS files are written to `~/.secure-kafka/tls/`:

```
~/.secure-kafka/tls/
  ca-cert                    # CA certificate
  ca-key                     # CA private key
  broker-1.keystore.p12      # Broker 1 keystore (cert + private key)
  broker-2.keystore.p12      # Broker 2 keystore
  broker-3.keystore.p12      # Broker 3 keystore
  broker.truststore.p12      # Shared truststore (contains CA cert)
```

For prod deployments, copy each broker's keystore and the shared truststore to the respective machine. The CA key should be kept secure and not deployed to broker machines.

### Customizing the scripts

Before running `0-all.sh`, you may need to edit the individual scripts:

**Custom domains or IPs** -- Edit the SAN (Subject Alternative Name) list in `2-generate-keystore.sh` and `4-sign-certs.sh`. By default, the certs are valid for:
- `broker-{N}`, `broker-{N}.example.com`, `localhost`, `host.docker.internal`, `127.0.0.1`

Add your domain or public IP to both files. In `2-generate-keystore.sh`:
```
-ext "SAN=dns:$BROKER,...,dns:kafka.yourdomain.com,ip:203.0.113.10"
```
In `4-sign-certs.sh`:
```
DNS.5 = kafka.yourdomain.com
IP.2  = 203.0.113.10
```

**Passwords** -- The default password is `secret-pass` in all scripts. Change `KEYSTORE_PASSWORD` and `PRIV_KEY_PASSWORD` in `2-generate-keystore.sh`, `3-generate-csr.sh`, and `KEYSTORE_PASSWORD` in `5-import-ca-cert.sh`, `6-import-signed-cert.sh`, `7-create-truststore.sh`. Use the same passwords in your config file.

**Certificate validity** -- Default is 365 days. Change `VALIDITY_DAYS` in `1-generate-ca.sh`, `2-generate-keystore.sh`, and `4-sign-certs.sh`.

## Commands

```
msk <command> -config <path> -broker <all|id> [-topic <name>]
```

All commands require `-config` and `-broker`. The `-topic` flag is only used by `create-topic`.

### run-broker

Deploy Kafka broker containers.

```bash
# Local: deploy all 3 brokers on this machine
msk run-broker -config config.json -broker all

# Prod: deploy a single broker (run on each machine)
msk run-broker -config config.json -broker 1
```

If a container with the same name already exists, the tool prompts to remove it before creating a new one.

### create-topic

Create a topic. Uses `numPartitions` and `replicationFactor` from the cluster config.

```bash
msk create-topic -config config.json -broker 1 -topic orders
```

### setup-users

Create SCRAM-SHA-512 credentials for every client in the config. Idempotent -- re-running updates existing users.

```bash
msk setup-users -config config.json -broker 1
```

### setup-acls

Grant ACL permissions for every client based on their `acls` config. Idempotent -- adding an existing ACL is a no-op.

```bash
msk setup-acls -config config.json -broker 1
```

### check-health

Check cluster health: KRaft quorum, topics, SCRAM users, and ACLs. Warns if any client from the config is missing in the cluster.

```bash
msk check-health -config config.json -broker 1
```

## Config

### Local deployment example

All 3 brokers on one machine. Brokers reach each other via `host.docker.internal`. Clients connect via `localhost`.

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
      "truststorePath": "/home/you/.secure-kafka/tls/broker.truststore.p12",
      "truststorePassword": "secret-pass"
    },
    "brokers": [
      {
        "id": 1,
        "internalHost": "host.docker.internal",
        "externalHost": "localhost",
        "volumePath": "/home/you/.secure-kafka/volumes/broker-1",
        "tls": {
          "keystorePath": "/home/you/.secure-kafka/tls/broker-1.keystore.p12",
          "keystorePassword": "secret-pass",
          "keyPassword": "secret-pass"
        }
      },
      {
        "id": 2,
        "internalHost": "host.docker.internal",
        "externalHost": "localhost",
        "volumePath": "/home/you/.secure-kafka/volumes/broker-2",
        "tls": {
          "keystorePath": "/home/you/.secure-kafka/tls/broker-2.keystore.p12",
          "keystorePassword": "secret-pass",
          "keyPassword": "secret-pass"
        }
      },
      {
        "id": 3,
        "internalHost": "host.docker.internal",
        "externalHost": "localhost",
        "volumePath": "/home/you/.secure-kafka/volumes/broker-3",
        "tls": {
          "keystorePath": "/home/you/.secure-kafka/tls/broker-3.keystore.p12",
          "keystorePassword": "secret-pass",
          "keyPassword": "secret-pass"
        }
      }
    ],
    "clients": [
      {
        "username": "my-producer",
        "password": "producer-secret",
        "acls": [
          { "topic": "orders", "permissions": ["produce"] }
        ]
      },
      {
        "username": "my-consumer",
        "password": "consumer-secret",
        "acls": [
          { "topic": "orders", "permissions": ["consume"] }
        ]
      }
    ]
  }
}
```

### Prod deployment example

Each broker on a separate machine. Brokers reach each other via public IPs. Clients also connect via public IPs.

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
      "truststorePath": "/home/kafka/.secure-kafka/tls/broker.truststore.p12",
      "truststorePassword": "strong-password"
    },
    "brokers": [
      {
        "id": 1,
        "internalHost": "10.0.1.1",
        "externalHost": "10.0.1.1",
        "volumePath": "/home/kafka/.secure-kafka/volumes/broker-1",
        "tls": {
          "keystorePath": "/home/kafka/.secure-kafka/tls/broker-1.keystore.p12",
          "keystorePassword": "strong-password",
          "keyPassword": "strong-password"
        }
      },
      {
        "id": 2,
        "internalHost": "10.0.2.1",
        "externalHost": "10.0.2.1",
        "volumePath": "/home/kafka/.secure-kafka/volumes/broker-2",
        "tls": {
          "keystorePath": "/home/kafka/.secure-kafka/tls/broker-2.keystore.p12",
          "keystorePassword": "strong-password",
          "keyPassword": "strong-password"
        }
      },
      {
        "id": 3,
        "internalHost": "10.0.3.1",
        "externalHost": "10.0.3.1",
        "volumePath": "/home/kafka/.secure-kafka/volumes/broker-3",
        "tls": {
          "keystorePath": "/home/kafka/.secure-kafka/tls/broker-3.keystore.p12",
          "keystorePassword": "strong-password",
          "keyPassword": "strong-password"
        }
      }
    ],
    "clients": [
      {
        "username": "order-service",
        "password": "order-service-secret",
        "acls": [
          { "topic": "orders", "permissions": ["produce", "consume"] }
        ]
      }
    ]
  }
}
```

### Config reference

**Cluster**

| Field | Description |
|---|---|
| `cluster.id` | KRaft cluster ID. Generate once: `docker run --rm apache/kafka /opt/kafka/bin/kafka-storage.sh random-uuid`. Must be the same on every node. |
| `cluster.numPartitions` | Default partition count when creating a topic. |
| `cluster.replicationFactor` | Default replication factor when creating a topic. |
| `cluster.minInsyncReplicas` | Minimum replicas that must acknowledge a write (with `acks=all`). |

**TLS (cluster-wide)**

| Field | Description |
|---|---|
| `tls.truststorePath` | Absolute path to the shared truststore. |
| `tls.truststorePassword` | Truststore password. |

**Brokers**

| Field | Description |
|---|---|
| `id` | Unique broker ID (1-9). |
| `internalHost` | Address for inter-broker and controller traffic. Use `host.docker.internal` for local, public IP for prod. |
| `externalHost` | Address clients connect to. Use `localhost` for local, public IP for prod. |
| `volumePath` | Host directory for broker data. Each broker must have its own path. |
| `tls.keystorePath` | Absolute path to this broker's keystore. |
| `tls.keystorePassword` | Keystore password. Must match what was used in the TLS scripts. |
| `tls.keyPassword` | Private key password. Must match what was used in the TLS scripts. |

**Clients**

| Field | Description |
|---|---|
| `username` | SCRAM-SHA-512 username. |
| `password` | SCRAM-SHA-512 password. |
| `acls` | List of `{topic, permissions}` entries. |

| Permission | What it grants |
|---|---|
| `produce` | Write and Describe on the topic. Write and Describe on transactional IDs prefixed with the username. |
| `consume` | Read and Describe on the topic. Read on consumer groups prefixed with the username. |

## Architecture

### Listeners

Each broker runs 3 listeners:

| Listener | Container Port | Host Port | Protocol | Auth | Purpose |
|---|---|---|---|---|---|
| CONTROLLER | 9093 | `{id}9093` | SSL | mTLS | KRaft consensus |
| INTERNAL | 9094 | `{id}9094` | SSL | mTLS | Inter-broker replication |
| EXTERNAL | 9095 | `{id}9095` | SASL_SSL | SCRAM-SHA-512 | Client connections |

For example, broker 1 maps to host ports 19093/19094/19095, broker 2 to 29093/29094/29095.

### What the tool derives

These Kafka settings are computed from the config and don't need to be specified:

- `controller.quorum.voters` -- built from all brokers using `internalHost` and the controller port
- `super.users` -- `User:broker-{id}` for every broker (brokers bypass ACL checks)
- `ssl.principal.mapping.rules` -- extracts the CN from the certificate DN
- JAAS config -- generated and mounted into each container
- Container names -- `broker-{id}`
