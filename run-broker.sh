#!/bin/bash

set -euo pipefail

# ── Broker Identity ──────────────────────────────────────────
NODE_ID=1
BROKER_NAME="broker-${NODE_ID}"
PROCESS_ROLES="broker,controller"

# ── Cluster Nodes ────────────────────────────────────────────
# For a distributed cluster, set INTERNAL and EXTERNAL to the public IP or hostname.
# For a single-machine setup, use localhost for both.
BROKER_1_INTERNAL_HOST="broker1.example.com"
BROKER_1_EXTERNAL_HOST="broker1.example.com"
BROKER_2_INTERNAL_HOST="broker2.example.com"
BROKER_2_EXTERNAL_HOST="broker2.example.com"
BROKER_3_INTERNAL_HOST="broker3.example.com"
BROKER_3_EXTERNAL_HOST="broker3.example.com"

# ── Derived from NODE_ID (do not edit) ───────────────────────
INTERNAL_HOST_VAR="BROKER_${NODE_ID}_INTERNAL_HOST"
INTERNAL_HOST="${!INTERNAL_HOST_VAR}"
EXTERNAL_HOST_VAR="BROKER_${NODE_ID}_EXTERNAL_HOST"
EXTERNAL_HOST="${!EXTERNAL_HOST_VAR}"
CONTROLLER_QUORUM_VOTERS="1@${BROKER_1_INTERNAL_HOST}:9093,2@${BROKER_2_INTERNAL_HOST}:9093,3@${BROKER_3_INTERNAL_HOST}:9093"

# ── Cluster ID (same on every node) ──────────────────────────
# Generate once:
#   docker run --rm apache/kafka /opt/kafka/bin/kafka-storage.sh random-uuid
CLUSTER_ID="lVTckYCITwWSX_siiDBoaw"

# ── TLS ──────────────────────────────────────────────────────
SECRETS_DIR="/opt/kafka/secrets"
KEYSTORE_PASSWORD="secret-pass"
TRUSTSTORE_PASSWORD="secret-pass"
KEY_PASSWORD="secret-pass"

# ── Data ─────────────────────────────────────────────────────
DATA_DIR="/opt/kafka/data"

# ── JAAS config for SCRAM-SHA-512 ────────────────────────────
# The SCRAM JAAS property name contains hyphens (scram-sha-512),
# which can't be represented as an environment variable.
# So we mount a JAAS config file instead.
cat > "${SECRETS_DIR}/jaas.conf" <<'EOF'
KafkaServer {
    org.apache.kafka.common.security.scram.ScramLoginModule required;
};
EOF

# ── Run ──────────────────────────────────────────────────────
docker run -d \
  --name "${BROKER_NAME}" \
  --publish 19093:9093 \
  --publish 19094:9094 \
  --publish 19095:9095 \
  --restart unless-stopped \
  \
  -v "${SECRETS_DIR}/${BROKER_NAME}.keystore.p12:/etc/kafka/secrets/broker.keystore.p12:ro" \
  -v "${SECRETS_DIR}/kafka.truststore.p12:/etc/kafka/secrets/kafka.truststore.p12:ro" \
  -v "${SECRETS_DIR}/jaas.conf:/etc/kafka/secrets/jaas.conf:ro" \
  -v "${DATA_DIR}:/var/lib/kafka/data" \
  \
  -e CLUSTER_ID="${CLUSTER_ID}" \
  -e KAFKA_OPTS="-Djava.security.auth.login.config=/etc/kafka/secrets/jaas.conf" \
  \
  -e KAFKA_NODE_ID="${NODE_ID}" \
  -e KAFKA_PROCESS_ROLES="${PROCESS_ROLES}" \
  -e KAFKA_CONTROLLER_QUORUM_VOTERS="${CONTROLLER_QUORUM_VOTERS}" \
  \
  -e KAFKA_LISTENERS="CONTROLLER://:9093,INTERNAL://:9094,EXTERNAL://:9095" \
  -e KAFKA_ADVERTISED_LISTENERS="INTERNAL://${INTERNAL_HOST}:9094,EXTERNAL://${EXTERNAL_HOST}:9095" \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP="CONTROLLER:SSL,INTERNAL:SSL,EXTERNAL:SASL_SSL" \
  -e KAFKA_CONTROLLER_LISTENER_NAMES="CONTROLLER" \
  -e KAFKA_INTER_BROKER_LISTENER_NAME="INTERNAL" \
  \
  -e KAFKA_SSL_KEYSTORE_LOCATION="/etc/kafka/secrets/broker.keystore.p12" \
  -e KAFKA_SSL_KEYSTORE_PASSWORD="${KEYSTORE_PASSWORD}" \
  -e KAFKA_SSL_KEY_PASSWORD="${KEY_PASSWORD}" \
  -e KAFKA_SSL_TRUSTSTORE_LOCATION="/etc/kafka/secrets/kafka.truststore.p12" \
  -e KAFKA_SSL_TRUSTSTORE_PASSWORD="${TRUSTSTORE_PASSWORD}" \
  -e KAFKA_SSL_KEYSTORE_TYPE="PKCS12" \
  -e KAFKA_SSL_TRUSTSTORE_TYPE="PKCS12" \
  \
  -e KAFKA_LISTENER_NAME_CONTROLLER_SSL_CLIENT_AUTH="required" \
  -e KAFKA_LISTENER_NAME_INTERNAL_SSL_CLIENT_AUTH="required" \
  -e KAFKA_LISTENER_NAME_EXTERNAL_SSL_CLIENT_AUTH="none" \
  \
  -e KAFKA_SASL_ENABLED_MECHANISMS="SCRAM-SHA-512" \
  -e KAFKA_LISTENER_NAME_EXTERNAL_SASL_ENABLED_MECHANISMS="SCRAM-SHA-512" \
  \
  -e KAFKA_AUTHORIZER_CLASS_NAME="org.apache.kafka.metadata.authorizer.StandardAuthorizer" \
  -e KAFKA_SSL_PRINCIPAL_MAPPING_RULES='RULE:^CN=(.*?),OU=.*$/$$1/,DEFAULT' \
  -e KAFKA_SUPER_USERS="User:broker-1;User:broker-2;User:broker-3" \
  -e KAFKA_ALLOW_EVERYONE_IF_NO_ACL_FOUND="false" \
  \
  -e KAFKA_LOG_DIRS="/var/lib/kafka/data" \
  -e KAFKA_NUM_PARTITIONS="3" \
  -e KAFKA_DEFAULT_REPLICATION_FACTOR="3" \
  -e KAFKA_MIN_INSYNC_REPLICAS="2" \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR="3" \
  \
  apache/kafka:latest
