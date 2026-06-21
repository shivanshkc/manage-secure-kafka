#!/bin/bash

set -euo pipefail

# Directory where the signed cert and the broker keystore is located.
DIR=$HOME/NewPersonal/heimdall/kafka-deployment/out/tls
# Password for the keystore.
KEYSTORE_PASSWORD='secret-pass'
# Name of the broker for whom the cert and key will be generated.
if [ -z "${BROKER}" ]; then
    BROKER=broker-1
fi

keytool -keystore "$DIR/$BROKER.keystore.p12" \
    -alias  "$BROKER" \
    -import -file "$DIR/$BROKER.crt" \
    -storepass "$KEYSTORE_PASSWORD" -noprompt
