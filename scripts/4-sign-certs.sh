#!/bin/bash

set -euo pipefail

# Directory where the CA cert, key, and the CSR is located.
DIR=$HOME/.secure-kafka/tls
# Validity of the output signed certificate.
VALIDITY_DAYS=365
# Name of the broker for whom the cert and key will be generated.
if [ -z "${BROKER}" ]; then
    BROKER=broker-1
fi

# Domains and IPs for which the cert will be valid.
SAN_EXT="
[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${BROKER}
DNS.2 = ${BROKER}.example.com
DNS.3 = localhost
DNS.4 = host.docker.internal
IP.1  = 127.0.0.1
"

openssl x509 -req \
  -CA    "$DIR/ca-cert" \
  -CAkey "$DIR/ca-key" \
  -in    "$DIR/$BROKER.csr" \
  -out   "$DIR/$BROKER.crt" \
  -days  $VALIDITY_DAYS \
  -CAcreateserial \
  -extfile <(echo "$SAN_EXT") \
  -extensions v3_req
