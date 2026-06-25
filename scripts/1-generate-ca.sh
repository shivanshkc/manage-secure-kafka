#!/bin/bash

set -euo pipefail

# Directory where the CA cert and key will be created.
OUT_DIR=$HOME/.secure-kafka/tls
# Validity period for the CA's cert and key.
VALIDITY_DAYS=365

openssl req -new -x509 \
  -keyout "$OUT_DIR/ca-key" \
  -out    "$OUT_DIR/ca-cert" \
  -days   $VALIDITY_DAYS \
  -nodes  \
  -subj   "/C=SG/O=xrpscan/OU=heimdall/CN=heimdall-kafka-ca"
