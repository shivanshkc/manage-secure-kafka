#!/bin/bash

set -euo pipefail

# Directory where the CA cert and the broker keystore is located.
DIR=$HOME/NewPersonal/heimdall/kafka-deployment/out/tls
# Password for the keystore.
KEYSTORE_PASSWORD='secret-pass'
# Name of the broker in whose keystore the CA cert will be imported.
BROKER=kafka-broker-1

keytool -keystore "$DIR/$BROKER.keystore.p12" \
    -alias  CARoot \
    -import -file "$DIR/ca-cert" \
    -storepass "$KEYSTORE_PASSWORD" -noprompt
