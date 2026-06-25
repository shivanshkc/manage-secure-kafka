#!/bin/bash

set -euo pipefail

# Directory where the CA cert and the broker keystore is located.
DIR=$HOME/.secure-kafka/tls
# Password for the keystore.
KEYSTORE_PASSWORD='secret-pass'
# Name of the broker for whom the cert and key will be generated.
if [ -z "${BROKER}" ]; then
    BROKER=broker-1
fi

keytool -keystore "$DIR/$BROKER.keystore.p12" \
    -alias  CARoot \
    -import -file "$DIR/ca-cert" \
    -storepass "$KEYSTORE_PASSWORD" -noprompt
