#!/bin/bash

MONGODB_CLIENT="mongosh"
PARSED=(${MONGO_IMAGE//:/ })
MONGODB_VERSION=${PARSED[1]}
MONGODB_VENDOR=${PARSED[0]}

if [ "`echo ${MONGODB_VERSION} | cut -c 1`" = "4" ]; then
  MONGODB_CLIENT="mongo"
fi
if [ "`echo ${MONGODB_VERSION} | cut -c 1`" = "5" ] && [ ${MONGODB_VENDOR} == "percona/percona-server-mongodb" ]; then
  MONGODB_CLIENT="mongo"
fi

mkdir /tmp/mongodb1 /tmp/mongodb2
mongod --fork --logpath=/dev/null --profile=2 --replSet=rs0 --tlsMode=requireTLS --tlsCertificateKeyFile=/etc/tls/certificates/server.pem --tlsCAFile=/etc/tls/certificates/ca.crt --tlsClusterFile=/etc/tls/certificates/client.pem --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb1 --port=27022
mongod --fork --logpath=/dev/null --profile=2 --replSet=rs0 --tlsMode=requireTLS --tlsCertificateKeyFile=/etc/tls/certificates/server.pem --tlsCAFile=/etc/tls/certificates/ca.crt --tlsClusterFile=/etc/tls/certificates/client.pem --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb2 --port=27023
$MONGODB_CLIENT --port 27022 --tls --tlsCAFile=/etc/tls/certificates/ca.crt --tlsCertificateKeyFile=/etc/tls/certificates/client.pem --tlsAllowInvalidHostnames --eval "rs.initiate( { _id : 'rs0', members: [{ _id: 0, host: 'localhost:27022' }, { _id: 1, host: 'localhost:27023', priority: 0 }]})"
tail -f /dev/null
