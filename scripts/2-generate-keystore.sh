#!/bin/bash

set -euo pipefail

# Directory where the keystore will be generated.
OUT_DIR=$HOME/.secure-kafka/tls
# Passwords for the keystore and the broker's private key.
KEYSTORE_PASSWORD='secret-pass'
PRIV_KEY_PASSWORD='secret-pass'
# Validity period of the broker's self-signed cert.
VALIDITY_DAYS=365
# Name of the broker for whom the cert and key will be generated.
if [ -z "${BROKER}" ]; then
    BROKER=broker-1
fi

keytool -keystore "$OUT_DIR/$BROKER.keystore.p12" \
    -storetype PKCS12 \
    -alias    "$BROKER" \
    -keyalg   RSA \
    -validity $VALIDITY_DAYS \
    -genkey   \
    -storepass "$KEYSTORE_PASSWORD" \
    -keypass   "$PRIV_KEY_PASSWORD" \
    -dname "CN=$BROKER,OU=heimdall,O=xrpscan,C=SG" \
    -ext "SAN=dns:$BROKER,dns:$BROKER.example.com,dns:localhost,dns:host.docker.internal,ip:127.0.0.1"
