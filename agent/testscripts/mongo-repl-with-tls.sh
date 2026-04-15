#!/bin/bash

MONGODB_CLIENT="mongosh"
MONGODB_VENDOR="${MONGO_IMAGE%%:*}"
MONGODB_VERSION="${MONGO_IMAGE##*:}"

echo "Using MongoDB image: $MONGO_IMAGE"
echo "MongoDB version: $MONGODB_VERSION"
echo "MongoDB vendor: $MONGODB_VENDOR"

if [[ "$MONGODB_VENDOR" == "percona/percona-server-mongodb" && "$MONGODB_VERSION" == 5* ]]; then
  MONGODB_CLIENT="mongo"
fi

mkdir /tmp/mongodb1 /tmp/mongodb2
mongod --fork --logpath=/dev/null --profile=2 --replSet=rs0 --tlsMode=requireTLS --tlsCertificateKeyFile=/etc/tls/certificates/server.pem --tlsCAFile=/etc/tls/certificates/ca.crt --tlsClusterFile=/etc/tls/certificates/client.pem --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb1 --port=27022
mongod --fork --logpath=/dev/null --profile=2 --replSet=rs0 --tlsMode=requireTLS --tlsCertificateKeyFile=/etc/tls/certificates/server.pem --tlsCAFile=/etc/tls/certificates/ca.crt --tlsClusterFile=/etc/tls/certificates/client.pem --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb2 --port=27023
$MONGODB_CLIENT --port 27022 --tls --tlsCAFile=/etc/tls/certificates/ca.crt --tlsCertificateKeyFile=/etc/tls/certificates/client.pem --tlsAllowInvalidHostnames --eval "rs.initiate( { _id : 'rs0', members: [{ _id: 0, host: 'localhost:27022' }, { _id: 1, host: 'localhost:27023', priority: 0 }]})"
tail -f /dev/null
