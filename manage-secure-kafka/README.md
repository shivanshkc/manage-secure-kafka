# manage-secure-kafka

CLI tool to deploy a Kafka cluster with mTLS (inter-broker) and SCRAM-SHA-512 (client-broker) authentication.

Built for `apache/kafka` Docker image. Works with KRaft (no ZooKeeper).

## Usage

```bash
manage-secure-kafka run-broker   -config config.json -broker <all|id>
manage-secure-kafka setup-users  -config config.json
manage-secure-kafka setup-acls   -config config.json
```

### run-broker

Deploys one or more Kafka broker containers.

- `-broker all` -- deploys all brokers (local/dev use).
- `-broker <id>` -- deploys a single broker by ID (prod use, one broker per machine).

Idempotent: destroys existing container with the same name before creating a new one.

### setup-users

Creates SCRAM-SHA-512 credentials for each client in the config.

Auto-discovers a running broker by matching the `CLUSTER_ID` env var from config against running containers via the Docker API (`ContainerInspect` -> `Config.Env`). Runs `kafka-configs.sh` inside the matched container.

Idempotent: creating a user that already exists updates it.

### setup-acls

Sets up ACL rules for clients defined in the config.

Same broker discovery as `setup-users`. Runs `kafka-acls.sh` inside the matched container.

Idempotent: adding an existing ACL is a no-op.

## Config

```json
{
  "cluster": {
    "id": "lVTckYCITwWSX_siiDBoaw",
    "numPartitions": 3,
    "replicationFactor": 3,
    "minInsyncReplicas": 2
  },
  "tls": {
    "truststorePath": "/opt/kafka/secrets/broker.truststore.p12",
    "truststorePassword": "secret-pass"
  },
  "brokers": [
    {
      "id": 1,
      "roles": ["broker", "controller"],
      "internalHost": "10.0.1.1",
      "externalHost": "10.0.1.1",
      "dataVolumePath": "/opt/kafka/data/broker-1",
      "tls": {
        "keystorePath": "/opt/kafka/secrets/broker-1.keystore.p12",
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
    },
    {
      "username": "service-b",
      "password": "service-b-secret",
      "acls": [
        { "topic": "orders", "permissions": ["consume"] }
      ]
    }
  ]
}
```

### Cluster-level fields

| Field | Description |
|---|---|
| `cluster.id` | KRaft cluster ID. Generate with `kafka-storage.sh random-uuid`. Same on every node. |
| `cluster.numPartitions` | Default partition count for new topics. |
| `cluster.replicationFactor` | Default replication factor for new topics. |
| `cluster.minInsyncReplicas` | Minimum replicas that must ack a write when producer uses `acks=all`. |
| `tls.truststorePath` | Path to shared truststore (contains the CA cert). |
| `tls.truststorePassword` | Truststore password. |

### Per-broker fields

| Field | Description |
|---|---|
| `id` | Unique node ID. |
| `roles` | `["broker", "controller"]` for initial nodes, `["broker"]` for later additions. |
| `internalHost` | Address for inter-broker and controller traffic. |
| `externalHost` | Address for client traffic. |
| `dataVolumePath` | Host path mounted as Kafka's data volume. |
| `tls.keystorePath` | Path to this broker's keystore. |
| `tls.keystorePassword` | Keystore password. |
| `tls.keyPassword` | Private key password. |
| `ports` | Optional. Override default ports (controller: 9093, internal: 9094, external: 9095). Required for local deployments where multiple brokers share one machine. |

`ports` example:
```json
{ "controller": 19093, "internal": 19094, "external": 19095 }
```

### Clients

Each entry creates a SCRAM-SHA-512 user via `setup-users`. The `acls` array defines per-topic permissions used by `setup-acls`.

| ACL permission | Kafka operations granted |
|---|---|
| `produce` | Write, Describe on topic; Write, Create on transactional ID (client username as prefix) |
| `consume` | Read, Describe on topic; Read on consumer group (client username as prefix) |

## Listeners

| Listener | Default Port | Protocol | Auth | Purpose |
|---|---|---|---|---|
| CONTROLLER | 9093 | SSL | mTLS | KRaft consensus |
| INTERNAL | 9094 | SSL | mTLS | Inter-broker replication |
| EXTERNAL | 9095 | SASL_SSL | SCRAM-SHA-512 | Client connections |

## Derived by the tool

- `controller.quorum.voters` -- from brokers with `"controller"` in roles, using `internalHost` + controller port.
- `super.users` -- `User:broker-{id}` for every broker.
- `ssl.principal.mapping.rules` -- extracts CN from certificate DN.
- Container name -- `broker-{id}`.
- JAAS config -- generated and mounted automatically.
- Docker port mappings -- from `ports` (or defaults) to container-internal ports 9093/9094/9095.

## Tech

- Go with `github.com/docker/docker/client` (Docker SDK).
- Broker discovery via `ContainerInspect` -> `Config.Env` -> `CLUSTER_ID` match.
- `ContainerExecCreate` / `ContainerExecAttach` for running `kafka-configs.sh` and `kafka-acls.sh`.
