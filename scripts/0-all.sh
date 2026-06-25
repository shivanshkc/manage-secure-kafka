#!/bin/bash

set -euo pipefail

BASE_DIR="$HOME/.secure-kafka"
TLS_DIR="$BASE_DIR/tls"
VOLUMES_DIR="$BASE_DIR/volumes"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Remove running containers.
docker rm -f broker-1 broker-2 broker-3 || true
# Remove existing volumes.
rm -rf "$VOLUMES_DIR"
mkdir -p "$VOLUMES_DIR"
# Remove existing TLS data.
rm -rf "$TLS_DIR"
mkdir -p "$TLS_DIR"

export BROKER=broker-1
"$SCRIPT_DIR/1-generate-ca.sh"
"$SCRIPT_DIR/2-generate-keystore.sh"
"$SCRIPT_DIR/3-generate-csr.sh"
"$SCRIPT_DIR/4-sign-certs.sh"
"$SCRIPT_DIR/5-import-ca-cert.sh"
"$SCRIPT_DIR/6-import-signed-cert.sh"
"$SCRIPT_DIR/7-create-truststore.sh"

export BROKER=broker-2
"$SCRIPT_DIR/2-generate-keystore.sh"
"$SCRIPT_DIR/3-generate-csr.sh"
"$SCRIPT_DIR/4-sign-certs.sh"
"$SCRIPT_DIR/5-import-ca-cert.sh"
"$SCRIPT_DIR/6-import-signed-cert.sh"

export BROKER=broker-3
"$SCRIPT_DIR/2-generate-keystore.sh"
"$SCRIPT_DIR/3-generate-csr.sh"
"$SCRIPT_DIR/4-sign-certs.sh"
"$SCRIPT_DIR/5-import-ca-cert.sh"
"$SCRIPT_DIR/6-import-signed-cert.sh"
