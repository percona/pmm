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

# A primary and an arbiter with authentication enabled. The arbiter stores no users,
# and since MongoDB 8.1 it rejects buildInfo from unauthenticated clients.
echo "pmmagenttestkeyfile" > /tmp/keyfile
chmod 400 /tmp/keyfile

mkdir /tmp/mongodb1 /tmp/mongodb2
mongod --fork --logpath=/dev/null --replSet=rs1 --keyFile=/tmp/keyfile --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb1 --port=27024
mongod --fork --logpath=/dev/null --replSet=rs1 --keyFile=/tmp/keyfile --bind_ip=0.0.0.0 --dbpath=/tmp/mongodb2 --port=27025
$MONGODB_CLIENT --port 27024 --eval "rs.initiate( { _id : 'rs1', members: [{ _id: 0, host: 'localhost:27024' }, { _id: 1, host: 'localhost:27025', arbiterOnly: true }]})"
tail -f /dev/null
