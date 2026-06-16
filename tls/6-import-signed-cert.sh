#!/bin/bash

set -euo pipefail

# Directory where the signed cert and the broker keystore is located.
DIR=$HOME/NewPersonal/heimdall/kafka-deployment/out/tls
# Password for the keystore.
KEYSTORE_PASSWORD='secret-pass'
# Name of the broker whose signed cert is to be imported.
BROKER=kafka-broker-1

keytool -keystore "$DIR/$BROKER.keystore.p12" \
    -alias  "$BROKER" \
    -import -file "$DIR/$BROKER.crt" \
    -storepass "$KEYSTORE_PASSWORD" -noprompt
