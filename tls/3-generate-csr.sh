#!/bin/bash

set -euo pipefail

# Directory where the keystore is located and where the CSR will be created.
DIR=$HOME/NewPersonal/heimdall/kafka-deployment/out/tls
# Passwords for the keystore and the broker's private key.
KEYSTORE_PASSWORD='secret-pass'
PRIV_KEY_PASSWORD='secret-pass'
# Name of the broker for which the CSR will be generated.
BROKER=kafka-broker-1

keytool -keystore "$DIR/$BROKER.keystore.p12" \
    -alias   "$BROKER" \
    -certreq \
    -file    "$DIR/$BROKER.csr" \
    -storepass "$KEYSTORE_PASSWORD" -keypass "$PRIV_KEY_PASSWORD"
