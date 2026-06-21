#!/bin/bash

set -euo pipefail

# Remove running containers.
docker rm -f broker-1 broker-2 broker-3 || true
# Remove existing volumes.
rm -rf /Users/skuchcha/NewPersonal/heimdall/kafka-deployment/manage-secure-kafka/volumes
mkdir -p /Users/skuchcha/NewPersonal/heimdall/kafka-deployment/manage-secure-kafka/volumes
# Remove existing TLS data.
rm -rf /Users/skuchcha/NewPersonal/heimdall/kafka-deployment/out
mkdir -p /Users/skuchcha/NewPersonal/heimdall/kafka-deployment/out/tls

export BROKER=broker-1
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/1-generate-ca.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/2-generate-keystore.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/3-generate-csr.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/4-sign-certs.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/5-import-ca-cert.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/6-import-signed-cert.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/7-create-truststore.sh

export BROKER=broker-2
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/2-generate-keystore.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/3-generate-csr.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/4-sign-certs.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/5-import-ca-cert.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/6-import-signed-cert.sh

export BROKER=broker-3
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/2-generate-keystore.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/3-generate-csr.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/4-sign-certs.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/5-import-ca-cert.sh
/Users/skuchcha/NewPersonal/heimdall/kafka-deployment/tls/6-import-signed-cert.sh
