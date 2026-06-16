#!/bin/bash

set -euo pipefail

# Directory where the CA cert is located, and where the truststore will be created.
DIR=$HOME/NewPersonal/heimdall/kafka-deployment/out/tls
# Password for the keystore.
KEYSTORE_PASSWORD='secret-pass'

# The truststore will be the same for all brokers.
keytool -keystore "$DIR/kafka.truststore.p12" \
    -storetype PKCS12 \
    -alias  CARoot \
    -import -file "$DIR/ca-cert" \
    -storepass "$KEYSTORE_PASSWORD" -noprompt
