#!/usr/bin/env bash
# Generates database incidents for testing PMM Assistant investigations: bad queries, load, and problems that
# show only in logs, on MySQL, PostgreSQL and MongoDB. Runs on the Docker host of the test VM; every workload
# runs inside the database containers. Usage and the scenario list: README.md in this directory.
set -uo pipefail

MYSQL_PRIMARY=${MYSQL_PRIMARY:-ps_pmm_replication_8_4_1}
MYSQL_REPLICA=${MYSQL_REPLICA:-ps_pmm_replication_8_4_2}
PG=${PG:-pdpgsql_pmm_17_1}
PG_UNIT=${PG_UNIT:-postgresql@17-main}
MONGOS=${MONGOS:-mongos}
MONGO_ROOT=${MONGO_ROOT:-root:root}
# Test-only password of the generated users: shop_web carries the steady traffic, shop_app the incidents.
APP_PASSWORD=${APP_PASSWORD:-shop_app_pw}
STATE_DIR=${STATE_DIR:-/root/adre-issues}
TRUTH=$STATE_DIR/incidents.jsonl
# In-container directory for workload files and their pid files.
CDIR=/tmp/adre-issues

# Engines alternate, and incidents that raise an alert are spread through the cycle.
ROTATION=(
  mysql_replica_break pg_lock_waits mongo_collscans pg_connection_storm mysql_deadlocks mongo_stepdown
  pg_dead_tuples mysql_full_scans mongo_auth_failures mysql_connection_storm pg_temp_files mongo_big_sorts
  pg_rollbacks mysql_auth_failures mongo_timeouts pg_vacuum_blocked mysql_lock_waits mongo_write_storm
  pg_deadlocks pg_auth_failures host_cpu_saturation pg_seq_scans pg_checkpoint_storm pg_shm_exhausted
)

mkdir -p "$STATE_DIR"

now() { date -u +%FT%TZ; }
log() { printf '%s %s\n' "$(now)" "$*"; }

# my <container> [mysql args]: mysql as root inside the container.
my() {
  local c=$1
  shift
  docker exec -i "$c" bash -c 'exec mysql --defaults-extra-file=<(printf "[client]\nuser=root\npassword=%s\n" "$MYSQL_ROOT_PASSWORD") -N -s "$@"' _ "$@"
}

pg() { docker exec -i "$PG" psql -U postgres -X -q -t -A -v ON_ERROR_STOP=1 "$@"; }

# mg <container> [mongosh args]: mongosh as root against the mongod/mongos in the container.
mg() {
  local c=$1
  shift
  docker exec -i "$c" mongosh --quiet "mongodb://$MONGO_ROOT@127.0.0.1:27017/admin" "$@"
}

app_uri() { echo "mongodb://$1:$APP_PASSWORD@127.0.0.1:27017/shop?authSource=shop"; }

# put <container> <file>: writes stdin to $CDIR/<file> in the container.
put() { docker exec -i "$1" bash -c "mkdir -p $CDIR && cat > $CDIR/$2"; }

# job <container> <name>: copies the bash script on stdin into the container and starts it in its own process group.
job() {
  local c=$1 name=$2
  put "$c" "$name.sh"
  docker exec "$c" bash -c "cd $CDIR; set -m; nohup bash $CDIR/$name.sh </dev/null >$CDIR/$name.log 2>&1 & echo \$! >$CDIR/$name.pid"
}

# stop_jobs <container> <name prefix>: kills the process groups of matching jobs.
stop_jobs() {
  docker exec "$1" bash -c "for f in $CDIR/$2*.pid; do [ -e \"\$f\" ] || continue; kill -TERM -- -\$(cat \"\$f\") 2>/dev/null; rm -f \"\$f\"; done"
}

running_jobs() {
  docker exec "$1" bash -c "for f in $CDIR/*.pid; do [ -e \"\$f\" ] || continue; kill -0 -- -\$(cat \"\$f\") 2>/dev/null && basename \"\$f\" .pid; done" 2>/dev/null | tr '\n' ' '
}

kill_mysql_app() {
  local id
  for id in $(my "$MYSQL_PRIMARY" -e "SELECT id FROM information_schema.processlist WHERE user = 'shop_app'" </dev/null); do
    my "$MYSQL_PRIMARY" -e "KILL $id" </dev/null 2>/dev/null
  done
}

kill_pg_app() { pg -d postgres -c "SELECT count(pg_terminate_backend(pid)) FROM pg_stat_activity WHERE usename = 'shop_app'" >/dev/null </dev/null; }

truth() {
  python3 -c 'import json, sys; print(json.dumps(dict(zip(["scenario", "start", "end", "target", "expect"], sys.argv[1:]))))' "$@" >>"$TRUTH"
}

# lua <name>: writes a sysbench script whose event() body is on stdin; init runs once per connection.
lua() {
  local name=$1 init=${2:-}
  {
    printf 'function thread_init()\n  drv = sysbench.sql.driver()\n  con = drv:connect()\n  %s\nend\n' "$init"
    printf 'function thread_done()\n  con:disconnect()\nend\n'
    printf 'function event()\n'
    cat
    printf 'end\n'
  } | put "$MYSQL_PRIMARY" "$name.lua"
}

# sb <name> <user> <threads> <seconds, 0 = forever> <rate, 0 = unthrottled>: runs $CDIR/<name>.lua on the MySQL primary.
sb() {
  local name=$1 user=$2 threads=$3 secs=$4 rate=$5
  job "$MYSQL_PRIMARY" "$name" <<EOF
while true; do
  sysbench $CDIR/$name.lua --db-driver=mysql --mysql-host=127.0.0.1 --mysql-user=$user --mysql-password=$APP_PASSWORD \
    --mysql-db=shop --mysql-ignore-errors=all --threads=$threads --time=$secs --rate=$rate --report-interval=0 run
  [ $secs -gt 0 ] && break
  sleep 5
done
EOF
}

# pgb <name> <user> <pgbench args>: runs pgbench against the shop database; scripts are in $CDIR.
# PGB_PGOPTIONS, when set, becomes PGOPTIONS of its sessions.
pgb() {
  local name=$1 user=$2
  shift 2
  job "$PG" "$name" <<EOF
export PGPASSWORD=$APP_PASSWORD PGOPTIONS='${PGB_PGOPTIONS:-}'
cd $CDIR
exec pgbench -h 127.0.0.1 -U $user -n $* shop
EOF
}

# mjs <container> <name> <processes> <uri> <seconds>: runs the mongosh script on stdin in parallel; END is defined.
mjs() {
  local c=$1 name=$2 procs=$3 uri=$4 secs=$5
  { printf 'const END = Date.now() + %d * 1000;\n' "$secs"; cat; } | put "$c" "$name.js"
  job "$c" "$name" <<EOF
for i in \$(seq $procs); do mongosh --quiet "$uri" $CDIR/$name.js & done
wait
EOF
}

mongo_shop_shard() { mg "$MONGOS" --eval 'print(db.getSiblingDB("config").databases.findOne({_id: "shop"}).primary)' </dev/null; }

# Shard members are containers named <shard>01..03 (rs1 -> rs101).
mongo_primary() { mg "${1}01" --eval 'print(db.hello().primary.split(":")[0])' </dev/null; }

### Setup and teardown

setup_mysql() {
  log "MySQL: settings, users and shop schema on $MYSQL_PRIMARY"
  my "$MYSQL_PRIMARY" <<SQL
SET PERSIST log_error_verbosity = 3;
SET PERSIST innodb_print_all_deadlocks = ON;
SET PERSIST max_connections = 200;
CREATE DATABASE IF NOT EXISTS shop;
CREATE USER IF NOT EXISTS 'shop_web'@'%' IDENTIFIED BY '$APP_PASSWORD';
CREATE USER IF NOT EXISTS 'shop_app'@'%' IDENTIFIED BY '$APP_PASSWORD';
GRANT SELECT, INSERT, UPDATE, DELETE ON shop.* TO 'shop_web'@'%', 'shop_app'@'%';
CREATE TABLE IF NOT EXISTS shop.customers (
  id INT PRIMARY KEY, email VARCHAR(100), name VARCHAR(100), country CHAR(2), created_at DATETIME);
CREATE TABLE IF NOT EXISTS shop.orders (
  id INT AUTO_INCREMENT PRIMARY KEY, customer_id INT NOT NULL, status VARCHAR(16), amount DECIMAL(10,2),
  created_at DATETIME, note VARCHAR(200));
CREATE TABLE IF NOT EXISTS shop.accounts (id INT PRIMARY KEY, balance BIGINT NOT NULL);
INSERT IGNORE INTO shop.accounts VALUES (1,1000),(2,1000),(3,1000),(4,1000),(5,1000),(6,1000),(7,1000),(8,1000),(9,1000),(10,1000);
SQL
  my "$MYSQL_REPLICA" -e "SET PERSIST log_error_verbosity = 3" </dev/null
  if [[ $(my "$MYSQL_PRIMARY" -e "SELECT COUNT(*) FROM shop.customers" </dev/null) -lt 200000 ]]; then
    log "MySQL: loading 200k customers"
    my "$MYSQL_PRIMARY" <<'SQL'
SET SESSION cte_max_recursion_depth = 300000;
INSERT IGNORE INTO shop.customers
WITH RECURSIVE s(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM s WHERE n < 200000)
SELECT n, CONCAT('user', n, '@example.com'), CONCAT('Customer ', n), ELT(1 + n % 5, 'US', 'DE', 'FR', 'BR', 'IN'),
       NOW() - INTERVAL n MINUTE FROM s;
SQL
  fi
  local orders
  orders=$(my "$MYSQL_PRIMARY" -e "SELECT COUNT(*) FROM shop.orders" </dev/null)
  while [[ $orders -lt 2000000 ]]; do
    log "MySQL: loading orders ($orders so far)"
    my "$MYSQL_PRIMARY" <<'SQL'
INSERT INTO shop.orders (customer_id, status, amount, created_at, note)
SELECT 1 + FLOOR(RAND() * 200000), ELT(1 + FLOOR(RAND() * 4), 'new', 'paid', 'shipped', 'cancelled'),
       ROUND(RAND() * 500, 2), NOW() - INTERVAL FLOOR(RAND() * 525600) MINUTE, CONCAT(MD5(RAND()), ' ', MD5(RAND()))
FROM shop.customers;
SQL
    orders=$(my "$MYSQL_PRIMARY" -e "SELECT COUNT(*) FROM shop.orders" </dev/null)
  done
}

setup_pg() {
  log "PostgreSQL: settings, roles and shop database on $PG"
  pg -d postgres -v pw="$APP_PASSWORD" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', r, :'pw') FROM unnest(ARRAY['shop_app', 'shop_web']) r
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = r) \gexec
SELECT 'CREATE DATABASE shop OWNER shop_app' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'shop') \gexec
ALTER SYSTEM SET max_connections = 200;
ALTER SYSTEM SET log_lock_waits = on;
ALTER SYSTEM SET log_temp_files = 0;
ALTER SYSTEM SET log_min_duration_statement = 1000;
SELECT pg_reload_conf() \g /dev/null
SQL
  log "PostgreSQL: loading tables (skipped when present)"
  pg -d shop <<'SQL'
SET client_min_messages = warning;
SET ROLE shop_app;
CREATE TABLE IF NOT EXISTS customers (id int PRIMARY KEY, email text, name text, country text, created_at timestamptz);
INSERT INTO customers
SELECT n, 'user' || n || '@example.com', 'Customer ' || n, (ARRAY['US', 'DE', 'FR', 'BR', 'IN'])[1 + n % 5],
       now() - n * interval '1 minute'
FROM generate_series(1, 200000) n WHERE NOT EXISTS (SELECT FROM customers);
CREATE TABLE IF NOT EXISTS orders (
  id bigserial PRIMARY KEY, customer_id int NOT NULL, status text, amount numeric(10,2), created_at timestamptz, note text);
INSERT INTO orders (customer_id, status, amount, created_at, note)
SELECT 1 + (random() * 199999)::int, (ARRAY['new', 'paid', 'shipped', 'cancelled'])[1 + (random() * 3)::int],
       round((random() * 500)::numeric, 2), now() - random() * interval '365 days',
       md5(random()::text) || ' ' || md5(random()::text)
FROM generate_series(1, 2000000) WHERE NOT EXISTS (SELECT FROM orders);
CREATE TABLE IF NOT EXISTS accounts (id int PRIMARY KEY, balance bigint NOT NULL);
INSERT INTO accounts SELECT n, 1000 FROM generate_series(1, 10) n ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS order_events (id int PRIMARY KEY, order_id bigint, kind text, n int) WITH (autovacuum_enabled = false);
INSERT INTO order_events SELECT n, n, 'created', 0 FROM generate_series(1, 300000) n ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS cart_items (id int PRIMARY KEY, cart_id int, sku text, qty int) WITH (log_autovacuum_min_duration = 0);
INSERT INTO cart_items SELECT n, n / 3, 'SKU-' || n % 5000, 1 FROM generate_series(1, 300000) n ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS wal_churn (id int, payload text);
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO shop_web;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO shop_web;
ANALYZE customers, orders, accounts, order_events, cart_items;
SQL
  if [[ $(pg -c "SHOW max_connections" </dev/null) != 200 ]]; then
    local start
    start=$(now)
    log "PostgreSQL: restarting $PG_UNIT to apply max_connections = 200"
    docker exec "$PG" systemctl restart "$PG_UNIT"
    truth setup_pg_restart "$start" "$(now)" "$PG" "Planned restart by issues.sh setup to set max_connections = 200; PostgreSQL restarted alert expected."
  fi
}

setup_mongo() {
  log "MongoDB: users and shop.orders through $MONGOS"
  { printf 'const PW = %s;\n' "$(python3 -c 'import json, sys; print(json.dumps(sys.argv[1]))' "$APP_PASSWORD")"; cat <<'JS'; } | put "$MONGOS" setup.js
const shop = db.getSiblingDB('shop');
for (const u of ['shop_app', 'shop_web']) {
  if (!shop.getUser(u)) shop.createUser({user: u, pwd: PW, roles: [{role: 'readWrite', db: 'shop'}]});
}
shop.orders.createIndex({order_no: 1}, {unique: true});
const statuses = ['new', 'paid', 'shipped', 'cancelled'];
const hex = () => Math.floor(Math.random() * 4294967296).toString(16).padStart(8, '0');
const item = () => ({sku: 'SKU-' + Math.floor(Math.random() * 5000), qty: 1 + Math.floor(Math.random() * 5), price: Math.round(Math.random() * 10000) / 100});
let n = shop.orders.countDocuments();
while (n < 1000000) {
  const docs = [];
  for (let i = 0; i < 5000; i++) {
    n++;
    docs.push({order_no: n, customer_id: 1 + Math.floor(Math.random() * 200000), status: statuses[Math.floor(Math.random() * 4)],
      amount: Math.round(Math.random() * 50000) / 100, created_at: new Date(Date.now() - Math.random() * 31536000000),
      note: hex() + hex() + hex() + hex() + ' ' + hex() + hex() + hex() + hex(), items: [item(), item(), item()]});
  }
  try { shop.orders.insertMany(docs, {ordered: false}); } catch (e) { print(e.message.slice(0, 200)); }
}
if (!shop.getCollectionNames().includes('audit_log')) shop.createCollection('audit_log', {capped: true, size: 256 * 1024 * 1024});
print('shop.orders: ' + shop.orders.estimatedDocumentCount() + ' docs, primary shard ' + db.getSiblingDB('config').databases.findOne({_id: 'shop'}).primary);
JS
  mg "$MONGOS" "$CDIR/setup.js" </dev/null
}

cmd_setup() {
  setup_mysql
  setup_pg
  setup_mongo
  log "setup done"
}

cmd_teardown() {
  cmd_stop
  log "removing shop data and restoring settings"
  my "$MYSQL_PRIMARY" <<'SQL'
DROP DATABASE IF EXISTS shop;
DROP USER IF EXISTS 'shop_web'@'%', 'shop_app'@'%';
RESET PERSIST IF EXISTS log_error_verbosity;
RESET PERSIST IF EXISTS innodb_print_all_deadlocks;
RESET PERSIST IF EXISTS max_connections;
SET GLOBAL log_error_verbosity = 2;
SET GLOBAL innodb_print_all_deadlocks = OFF;
SET GLOBAL max_connections = 1000;
SQL
  my "$MYSQL_REPLICA" -e "RESET PERSIST IF EXISTS log_error_verbosity; SET GLOBAL log_error_verbosity = 2" </dev/null
  pg -d postgres <<'SQL'
DROP DATABASE IF EXISTS shop WITH (FORCE);
DROP ROLE IF EXISTS shop_app;
DROP ROLE IF EXISTS shop_web;
ALTER SYSTEM RESET max_connections;
ALTER SYSTEM RESET log_lock_waits;
ALTER SYSTEM RESET log_temp_files;
ALTER SYSTEM RESET log_min_duration_statement;
ALTER SYSTEM RESET max_wal_size;
ALTER SYSTEM RESET min_wal_size;
SQL
  docker exec "$PG" systemctl restart "$PG_UNIT"
  mg "$MONGOS" --eval 'const s = db.getSiblingDB("shop"); s.dropAllUsers(); s.dropDatabase();' </dev/null
  log "teardown done"
}

### Steady traffic

cmd_baseline_start() {
  log "starting steady traffic (shop_web) on MySQL, PostgreSQL and MongoDB"
  lua baseline-mysql <<'LUA'
  local r = sysbench.rand.uniform(1, 10)
  if r <= 7 then
    con:query(string.format("SELECT id, customer_id, status, amount FROM orders WHERE id = %d", sysbench.rand.uniform(1, 2000000)))
  elseif r <= 9 then
    con:query(string.format("UPDATE orders SET status = 'shipped' WHERE id = %d", sysbench.rand.uniform(1, 2000000)))
  else
    con:query(string.format("INSERT INTO orders (customer_id, status, amount, created_at, note) VALUES (%d, 'new', %d, NOW(), 'web')",
      sysbench.rand.uniform(1, 200000), sysbench.rand.uniform(1, 500)))
  end
LUA
  sb baseline-mysql shop_web 2 0 15

  put "$PG" baseline-read.sql <<'SQL'
\set id random(1, 2000000)
SELECT id, customer_id, status, amount FROM orders WHERE id = :id;
SQL
  put "$PG" baseline-update.sql <<'SQL'
\set id random(1, 2000000)
UPDATE orders SET status = 'shipped' WHERE id = :id;
SQL
  put "$PG" baseline-insert.sql <<'SQL'
\set c random(1, 200000)
INSERT INTO orders (customer_id, status, amount, created_at, note) VALUES (:c, 'new', 10, now(), 'web');
SQL
  job "$PG" baseline-pg <<EOF
export PGPASSWORD=$APP_PASSWORD
cd $CDIR
while true; do
  pgbench -h 127.0.0.1 -U shop_web -n -c 2 -R 15 -T 86400 -f baseline-read.sql@7 -f baseline-update.sql@2 -f baseline-insert.sql@1 shop
  sleep 5
done
EOF

  { printf 'const END = Date.now() + 10 * 365 * 86400 * 1000;\n'; cat <<'JS'; } | put "$MONGOS" baseline-mongo.js
const s = db.getSiblingDB('shop');
while (Date.now() < END) {
  const r = Math.random(), no = 1 + Math.floor(Math.random() * 1000000);
  if (r < 0.7) s.orders.findOne({order_no: no});
  else if (r < 0.9) s.orders.updateOne({order_no: no}, {$set: {status: 'shipped'}});
  else s.audit_log.insertOne({at: new Date(), order_no: no, event: 'viewed'});
  sleep(60);
}
JS
  job "$MONGOS" baseline-mongo <<EOF
while true; do mongosh --quiet "$(app_uri shop_web)" $CDIR/baseline-mongo.js; sleep 5; done
EOF
}

cmd_baseline_stop() {
  stop_jobs "$MYSQL_PRIMARY" baseline
  stop_jobs "$PG" baseline
  stop_jobs "$MONGOS" baseline
}

### Scenarios. start_<name> <seconds> starts the workload, stop_<name> undoes it.

declare -A TARGET EXPECT

TARGET[mysql_full_scans]="MySQL $MYSQL_PRIMARY, shop.orders"
EXPECT[mysql_full_scans]="QAN: SELECT on shop.orders by customer_id, note LIKE '%..%' and a join to customers examine ~2M rows per call with no index (full table scan); threads_running and CPU rise. Cause: orders.customer_id has no index and leading-wildcard LIKE cannot use one. Fix: CREATE INDEX on orders(customer_id); avoid LIKE '%x%'."
start_mysql_full_scans() {
  lua mysql_full_scans <<'LUA'
  local r = sysbench.rand.uniform(1, 3)
  if r == 1 then
    con:query(string.format("SELECT COUNT(*), SUM(amount) FROM orders WHERE customer_id = %d", sysbench.rand.uniform(1, 200000)))
  elseif r == 2 then
    con:query(string.format("SELECT id, customer_id, amount FROM orders WHERE note LIKE '%%%04x%%' LIMIT 20", sysbench.rand.uniform(0, 65535)))
  else
    con:query("SELECT c.country, COUNT(*), SUM(o.amount) FROM orders o JOIN customers c ON c.id = o.customer_id WHERE o.status = 'paid' GROUP BY c.country ORDER BY 3 DESC")
  end
LUA
  sb mysql_full_scans shop_app "${2:-8}" "$1" 0
}
stop_mysql_full_scans() {
  stop_jobs "$MYSQL_PRIMARY" mysql_full_scans
  kill_mysql_app
}

TARGET[mysql_lock_waits]="MySQL $MYSQL_PRIMARY, shop.orders"
EXPECT[mysql_lock_waits]="One shop_app transaction runs UPDATE orders ... WHERE customer_id = 4242 (no index, so it locks every row) and stays open in SELECT SLEEP; other UPDATEs on orders wait and fail with 1205 Lock wait timeout. QAN: UPDATE orders with high lock time and errors; InnoDB row lock waits/time rise; processlist and sys.innodb_lock_waits show the blocker. Fix: kill the blocking transaction, index customer_id, keep transactions short."
start_mysql_lock_waits() {
  local secs=$1
  job "$MYSQL_PRIMARY" mysql_lock_waits-holder <<EOF
mysql -h127.0.0.1 -ushop_app -p$APP_PASSWORD shop -e "BEGIN; UPDATE orders SET status = 'review' WHERE customer_id = 4242; SELECT SLEEP($secs); ROLLBACK;"
EOF
  lua mysql_lock_waits "con:query('SET SESSION innodb_lock_wait_timeout = 10')" <<'LUA'
  con:query(string.format("UPDATE orders SET status = 'paid' WHERE id = %d", sysbench.rand.uniform(1, 2000000)))
LUA
  sb mysql_lock_waits shop_app 6 "$secs" 0
}
stop_mysql_lock_waits() {
  stop_jobs "$MYSQL_PRIMARY" mysql_lock_waits
  kill_mysql_app
}

TARGET[mysql_deadlocks]="MySQL $MYSQL_PRIMARY, shop.accounts"
EXPECT[mysql_deadlocks]="Error log: repeated 'Transactions deadlock detected' dumps (innodb_print_all_deadlocks) naming shop.accounts and two UPDATE accounts statements; InnoDB deadlock counter rises; QAN: UPDATE accounts with 1213 errors. Cause: transfers update two account rows in inconsistent order. Fix: lock rows in a consistent order (lower id first) or SELECT ... FOR UPDATE both rows first."
start_mysql_deadlocks() {
  lua mysql_deadlocks <<'LUA'
  local a = sysbench.rand.uniform(1, 10)
  local b = sysbench.rand.uniform(1, 10)
  if a == b then b = a % 10 + 1 end
  con:query("BEGIN")
  con:query(string.format("UPDATE accounts SET balance = balance - 1 WHERE id = %d", a))
  con:query("DO SLEEP(0.05)")
  con:query(string.format("UPDATE accounts SET balance = balance + 1 WHERE id = %d", b))
  con:query("COMMIT")
LUA
  sb mysql_deadlocks shop_app 8 "$1" 30
}
stop_mysql_deadlocks() {
  stop_jobs "$MYSQL_PRIMARY" mysql_deadlocks
  kill_mysql_app
}

TARGET[mysql_connection_storm]="MySQL $MYSQL_PRIMARY"
EXPECT[mysql_connection_storm]="Alert 'MySQL connections near max_connections': ~180 of max_connections 200 used by shop_app sessions sitting in SELECT SLEEP(30) from 127.0.0.1; every 30 s a burst of extra clients is refused (Too many connections, Connection_errors_max_connections rises). Cause: a connection pool larger than max_connections / leaked connections. Fix: smaller pool or a proxy, then max_connections."
start_mysql_connection_storm() {
  local secs=$1
  lua mysql_connection_storm <<'LUA'
  con:query("SELECT SLEEP(30)")
LUA
  sb mysql_connection_storm shop_app 175 "$secs" 0
  # Bursts start once the long connections are open; otherwise those fail to connect and the client exits.
  job "$MYSQL_PRIMARY" mysql_connection_storm-burst <<EOF
sleep 20
end=\$((SECONDS + $secs - 20))
while [ \$SECONDS -lt \$end ]; do
  for i in \$(seq 30); do mysql -h127.0.0.1 -ushop_app -p$APP_PASSWORD shop -e "SELECT SLEEP(3)" >/dev/null 2>&1 & done
  sleep 30
done
wait
EOF
}
stop_mysql_connection_storm() {
  stop_jobs "$MYSQL_PRIMARY" mysql_connection_storm
  kill_mysql_app
}

TARGET[mysql_auth_failures]="MySQL $MYSQL_PRIMARY (error log only)"
EXPECT[mysql_auth_failures]="Error log only: repeated Notes 'Access denied for user 'shop_app'@'127.0.0.1' (using password: YES)' (MY-010926) and the same for the unknown user report_ro, about once a second each; Aborted_connects / access denied counters rise; no alert. Cause: a client deployed with a wrong password (or a brute-force attempt) from 127.0.0.1."
start_mysql_auth_failures() {
  job "$MYSQL_PRIMARY" mysql_auth_failures <<EOF
end=\$((SECONDS + $1))
while [ \$SECONDS -lt \$end ]; do
  mysql -h127.0.0.1 -ushop_app -pwinter2024 shop -e "SELECT 1" >/dev/null 2>&1
  mysql -h127.0.0.1 -ureport_ro -preport shop -e "SELECT 1" >/dev/null 2>&1
  sleep 1
done
EOF
}
stop_mysql_auth_failures() { stop_jobs "$MYSQL_PRIMARY" mysql_auth_failures; }

TARGET[mysql_replica_break]="MySQL replica $MYSQL_REPLICA"
EXPECT[mysql_replica_break]="Alert 'MySQL replication SQL thread stopped' on the replica; replica error log: applier error 1062 Duplicate entry '900001' for key 'customers.PRIMARY' (MY-010584 and the coordinator stop message); Seconds_Behind_Source NULL, relay log grows. Cause: a row written directly on the replica (read_only=OFF) conflicts with the same row from the source. Fix: delete the conflicting row on the replica (or skip that GTID), START REPLICA, set super_read_only=ON on replicas."
start_mysql_replica_break() {
  my "$MYSQL_REPLICA" <<'SQL'
SET SESSION sql_log_bin = 0;
INSERT INTO shop.customers VALUES (900001, 'drift@example.com', 'Replica drift', 'US', NOW());
SQL
  my "$MYSQL_PRIMARY" -e "INSERT INTO shop.customers VALUES (900001, 'new@example.com', 'New customer', 'DE', NOW())" </dev/null
}
stop_mysql_replica_break() {
  my "$MYSQL_REPLICA" <<'SQL'
SET SESSION sql_log_bin = 0;
DELETE FROM shop.customers WHERE id = 900001;
START REPLICA;
SQL
  sleep 5
  my "$MYSQL_PRIMARY" -e "DELETE FROM shop.customers WHERE id = 900001" </dev/null
}

TARGET[pg_seq_scans]="PostgreSQL $PG, shop.orders"
EXPECT[pg_seq_scans]="QAN (pg_stat_monitor): SELECTs on orders by customer_id, note LIKE and a join to customers read ~2M rows per call (Seq Scan, parallel workers); log 'duration: N ms' lines (log_min_duration_statement=1000); seq_scan count on orders rises; CPU up. Cause: missing index on orders(customer_id). Fix: CREATE INDEX CONCURRENTLY on orders(customer_id)."
start_pg_seq_scans() {
  put "$PG" pg_seq_scans-a.sql <<'SQL'
\set c random(1, 200000)
SELECT count(*), sum(amount) FROM orders WHERE customer_id = :c;
SQL
  put "$PG" pg_seq_scans-b.sql <<'SQL'
\set h random(0, 65535)
SELECT id, amount FROM orders WHERE note LIKE '%' || lpad(to_hex(:h), 4, '0') || '%' LIMIT 20;
SQL
  put "$PG" pg_seq_scans-c.sql <<'SQL'
SELECT c.country, count(*), sum(o.amount) FROM orders o JOIN customers c ON c.id = o.customer_id WHERE o.status = 'paid' GROUP BY c.country ORDER BY 3 DESC;
SQL
  local c=${2:-8}
  # Parallel hash joins overflow the container's 64 MB /dev/shm and abort pgbench clients (pg_shm_exhausted).
  PGB_PGOPTIONS='-c max_parallel_workers_per_gather=0' \
    pgb pg_seq_scans shop_app -c "$c" -j "$c" -T "$1" -f pg_seq_scans-a.sql -f pg_seq_scans-b.sql -f pg_seq_scans-c.sql
}
stop_pg_seq_scans() {
  stop_jobs "$PG" pg_seq_scans
  kill_pg_app
}

TARGET[pg_shm_exhausted]="PostgreSQL $PG (container /dev/shm)"
EXPECT[pg_shm_exhausted]="Logs: 'ERROR: could not resize shared memory segment \"/PostgreSQL.N\" to N bytes: No space left on device' with CONTEXT 'parallel worker', for the orders/customers join and GROUP BY run by shop_app; those queries fail, the same SQL without parallel workers works. Cause: the container has Docker's default 64 MB /dev/shm and parallel hash joins need more dynamic shared memory. Fix: start the container with a larger --shm-size (e.g. 1g), or lower max_parallel_workers_per_gather / work_mem."
start_pg_shm_exhausted() {
  job "$PG" pg_shm_exhausted <<EOF
export PGPASSWORD=$APP_PASSWORD PGOPTIONS='-c work_mem=64MB'
end=\$((SECONDS + $1))
for w in 1 2 3 4 5 6; do
  (while [ \$SECONDS -lt \$end ]; do
    psql -h 127.0.0.1 -U shop_app -d shop -c "SELECT c.country, o.status, count(*), sum(o.amount) FROM orders o JOIN customers c ON c.id = o.customer_id GROUP BY c.country, o.status" >/dev/null 2>&1
    sleep 2
  done) &
done
wait
EOF
}
stop_pg_shm_exhausted() {
  stop_jobs "$PG" pg_shm_exhausted
  kill_pg_app
}

TARGET[pg_lock_waits]="PostgreSQL $PG, shop.orders"
EXPECT[pg_lock_waits]="A shop_app session ran UPDATE orders ... WHERE id <= 20000 and sits 'idle in transaction'; other UPDATEs wait. Logs: 'process N still waiting for ShareLock on transaction X after 1000 ms' (log_lock_waits) with the blocking pid, then 'ERROR: canceling statement due to statement timeout'. pg_stat_activity shows the idle-in-transaction blocker. Fix: terminate it, set idle_in_transaction_session_timeout, commit promptly in the app."
start_pg_lock_waits() {
  local secs=$1
  job "$PG" pg_lock_waits-holder <<EOF
export PGPASSWORD=$APP_PASSWORD
(echo "BEGIN; UPDATE orders SET status = 'review' WHERE id <= 20000;"; sleep $secs; echo "ROLLBACK;") | psql -h 127.0.0.1 -U shop_app -d shop
EOF
  job "$PG" pg_lock_waits-waiters <<EOF
export PGPASSWORD=$APP_PASSWORD PGOPTIONS='-c statement_timeout=10s'
end=\$((SECONDS + $secs))
for w in 1 2 3 4; do
  (while [ \$SECONDS -lt \$end ]; do psql -h 127.0.0.1 -U shop_app -d shop -c "UPDATE orders SET amount = amount + 1 WHERE id = \$((RANDOM % 20000 + 1))" >/dev/null 2>&1; sleep 1; done) &
done
wait
EOF
}
stop_pg_lock_waits() {
  stop_jobs "$PG" pg_lock_waits
  kill_pg_app
}

TARGET[pg_deadlocks]="PostgreSQL $PG, shop.accounts"
EXPECT[pg_deadlocks]="Logs: 'ERROR: deadlock detected' with DETAIL 'Process N waits for ShareLock on transaction ...; blocked by process M' and the two UPDATE accounts statements; pg_stat_database deadlocks rise for db shop; rollbacks rise. Cause: transfers update accounts rows in inconsistent order. Fix: consistent lock order or SELECT ... FOR UPDATE ordered by id."
start_pg_deadlocks() {
  put "$PG" pg_deadlocks.sql <<'SQL'
\set a random(1, 10)
\set b random(1, 10)
BEGIN;
UPDATE accounts SET balance = balance - 1 WHERE id = :a;
SELECT pg_sleep(0.05);
UPDATE accounts SET balance = balance + 1 WHERE id = :b;
COMMIT;
SQL
  pgb pg_deadlocks shop_app -c 8 -j 8 -R 30 -T "$1" --max-tries=1 -f pg_deadlocks.sql
}
stop_pg_deadlocks() {
  stop_jobs "$PG" pg_deadlocks
  kill_pg_app
}

TARGET[pg_connection_storm]="PostgreSQL $PG"
EXPECT[pg_connection_storm]="Alert 'PostgreSQL connections near max_connections': ~175 shop_app sessions from 127.0.0.1 running pg_sleep(30) against max_connections 200; bursts every 30 s push past the limit, logging 'FATAL: sorry, too many clients already' (exporter scrapes may fail at those moments). Cause: client pool larger than max_connections, no pooler. Fix: pgbouncer / smaller pool."
start_pg_connection_storm() {
  local secs=$1
  put "$PG" pg_connection_storm.sql <<'SQL'
SELECT pg_sleep(30);
SQL
  pgb pg_connection_storm shop_app -c 175 -j 8 -T "$secs" -f pg_connection_storm.sql
  # Bursts start once the long connections are open; otherwise those fail to connect and the client exits.
  job "$PG" pg_connection_storm-burst <<EOF
export PGPASSWORD=$APP_PASSWORD
sleep 20
end=\$((SECONDS + $secs - 20))
while [ \$SECONDS -lt \$end ]; do
  for i in \$(seq 30); do psql -h 127.0.0.1 -U shop_app -d shop -c "SELECT pg_sleep(3)" >/dev/null 2>&1 & done
  sleep 30
done
wait
EOF
}
stop_pg_connection_storm() {
  stop_jobs "$PG" pg_connection_storm
  kill_pg_app
}

TARGET[pg_auth_failures]="PostgreSQL $PG (logs only)"
EXPECT[pg_auth_failures]="Logs only: repeated 'FATAL: password authentication failed for user \"shop_app\"' with DETAIL 'Connection matched file /etc/postgresql/17/main/pg_hba.conf line 2', and the same for the nonexistent role report_ro (DETAIL Role does not exist), about 2 per second from 127.0.0.1; no alert. Cause: client with a wrong password or brute force."
start_pg_auth_failures() {
  job "$PG" pg_auth_failures <<EOF
end=\$((SECONDS + $1))
while [ \$SECONDS -lt \$end ]; do
  PGPASSWORD=winter2024 psql -h 127.0.0.1 -U shop_app -d shop -c "SELECT 1" >/dev/null 2>&1
  PGPASSWORD=report psql -h 127.0.0.1 -U report_ro -d shop -c "SELECT 1" >/dev/null 2>&1
  sleep 1
done
EOF
}
stop_pg_auth_failures() { stop_jobs "$PG" pg_auth_failures; }

TARGET[pg_dead_tuples]="PostgreSQL $PG, shop.order_events"
EXPECT[pg_dead_tuples]="Alert 'PostgreSQL high number of dead tuples' on order_events: n_dead_tup climbs by 100k every minute and never drops; last_autovacuum stays empty; pg_class.reloptions shows autovacuum_enabled=false for order_events. Cause: autovacuum disabled on the table. Fix: ALTER TABLE order_events RESET (autovacuum_enabled); VACUUM order_events. (The scenario runs VACUUM at the end; the reloption stays.)"
start_pg_dead_tuples() {
  job "$PG" pg_dead_tuples <<EOF
export PGPASSWORD=$APP_PASSWORD
end=\$((SECONDS + $1))
while [ \$SECONDS -lt \$end ]; do
  psql -h 127.0.0.1 -U shop_app -d shop -c "UPDATE order_events SET n = n + 1, kind = 'updated' WHERE id <= 100000" >/dev/null
  sleep 60
done
EOF
}
stop_pg_dead_tuples() {
  stop_jobs "$PG" pg_dead_tuples
  pg -d shop -c "VACUUM order_events" </dev/null
}

TARGET[pg_vacuum_blocked]="PostgreSQL $PG, shop.cart_items"
EXPECT[pg_vacuum_blocked]="Alert 'PostgreSQL high number of dead tuples' on cart_items although autovacuum runs: autovacuum log lines for cart_items report tuples 'dead but not yet removable' and an old removable cutoff; a shop_app session has been 'idle in transaction' (REPEATABLE READ snapshot) since the start, holding back xmin (pg_stat_activity backend_xmin). Fix: terminate the idle transaction, set idle_in_transaction_session_timeout."
start_pg_vacuum_blocked() {
  local secs=$1
  job "$PG" pg_vacuum_blocked-holder <<EOF
export PGPASSWORD=$APP_PASSWORD
(echo "BEGIN ISOLATION LEVEL REPEATABLE READ; SELECT count(*) FROM customers;"; sleep $secs; echo "COMMIT;") | psql -h 127.0.0.1 -U shop_app -d shop
EOF
  job "$PG" pg_vacuum_blocked-churn <<EOF
export PGPASSWORD=$APP_PASSWORD
sleep 5
end=\$((SECONDS + $secs))
while [ \$SECONDS -lt \$end ]; do
  psql -h 127.0.0.1 -U shop_web -d shop -c "UPDATE cart_items SET qty = qty + 1 WHERE id <= 100000" >/dev/null
  sleep 60
done
EOF
}
stop_pg_vacuum_blocked() {
  stop_jobs "$PG" pg_vacuum_blocked
  kill_pg_app
  pg -d shop -c "VACUUM cart_items" </dev/null
}

TARGET[pg_rollbacks]="PostgreSQL $PG, database shop"
EXPECT[pg_rollbacks]="Alert 'PostgreSQL high transaction rollbacks' for db shop: rollbacks outnumber commits; logs 'ERROR: duplicate key value violates unique constraint \"customers_pkey\"' DETAIL Key (id)=(1) already exists, every second; QAN: UPDATE accounts transactions that end in ROLLBACK. Cause: an app retrying an insert of an existing id and a job that always rolls back."
start_pg_rollbacks() {
  put "$PG" pg_rollbacks.sql <<'SQL'
\set a random(1, 10)
BEGIN;
UPDATE accounts SET balance = balance WHERE id = :a;
ROLLBACK;
SQL
  pgb pg_rollbacks shop_app -c 4 -j 4 -R 40 -T "$1" -f pg_rollbacks.sql
  job "$PG" pg_rollbacks-dupes <<EOF
export PGPASSWORD=$APP_PASSWORD
end=\$((SECONDS + $1))
while [ \$SECONDS -lt \$end ]; do
  psql -h 127.0.0.1 -U shop_app -d shop -c "INSERT INTO customers (id, email, name, country, created_at) VALUES (1, 'dup@example.com', 'Duplicate', 'US', now())" >/dev/null 2>&1
  sleep 1
done
EOF
}
stop_pg_rollbacks() {
  stop_jobs "$PG" pg_rollbacks
  kill_pg_app
}

TARGET[pg_temp_files]="PostgreSQL $PG, role shop_app"
EXPECT[pg_temp_files]="Logs: 'temporary file: path \"base/pgsql_tmp/...\", size N' with the GROUP BY / ORDER BY statements on orders (log_temp_files=0); pg_stat_database temp_files/temp_bytes rise; QAN temp blocks written; queries slower. Cause: ALTER ROLE shop_app SET work_mem = '64kB' (pg_db_role_setting). Fix: ALTER ROLE shop_app RESET work_mem."
start_pg_temp_files() {
  pg -d postgres -c "ALTER ROLE shop_app SET work_mem = '64kB'" </dev/null
  put "$PG" pg_temp_files-a.sql <<'SQL'
SELECT customer_id, sum(amount) FROM orders GROUP BY customer_id ORDER BY 2 DESC LIMIT 10;
SQL
  put "$PG" pg_temp_files-b.sql <<'SQL'
\set o random(10000, 100000)
SELECT id, note FROM orders ORDER BY note LIMIT 10 OFFSET :o;
SQL
  pgb pg_temp_files shop_app -c 4 -j 4 -T "$1" -f pg_temp_files-a.sql -f pg_temp_files-b.sql
}
stop_pg_temp_files() {
  stop_jobs "$PG" pg_temp_files
  pg -d postgres -c "ALTER ROLE shop_app RESET work_mem" </dev/null
  kill_pg_app
}

TARGET[pg_checkpoint_storm]="PostgreSQL $PG"
EXPECT[pg_checkpoint_storm]="Logs: 'checkpoints are occurring too frequently (N seconds apart)' with HINT 'Consider increasing the configuration parameter max_wal_size', plus 'checkpoint starting: wal' lines (log_checkpoints); requested checkpoints and WAL volume climb; writes slow. Cause: max_wal_size = 32MB set with ALTER SYSTEM during bulk insert/delete of wal_churn. Fix: raise max_wal_size (ALTER SYSTEM RESET max_wal_size)."
start_pg_checkpoint_storm() {
  pg -d postgres <<'SQL'
ALTER SYSTEM SET min_wal_size = '32MB';
ALTER SYSTEM SET max_wal_size = '32MB';
SELECT pg_reload_conf() \g /dev/null
SQL
  job "$PG" pg_checkpoint_storm <<EOF
export PGPASSWORD=$APP_PASSWORD
end=\$((SECONDS + $1))
while [ \$SECONDS -lt \$end ]; do
  psql -h 127.0.0.1 -U shop_app -d shop -c "INSERT INTO wal_churn SELECT g, md5(g::text) || md5(random()::text) FROM generate_series(1, 300000) g" -c "DELETE FROM wal_churn" >/dev/null
  sleep 2
done
EOF
}
stop_pg_checkpoint_storm() {
  stop_jobs "$PG" pg_checkpoint_storm
  pg -d postgres <<'SQL'
ALTER SYSTEM RESET min_wal_size;
ALTER SYSTEM RESET max_wal_size;
SELECT pg_reload_conf() \g /dev/null
SQL
  pg -d shop -c "TRUNCATE wal_churn" </dev/null
}

TARGET[mongo_collscans]="MongoDB shop.orders via $MONGOS (shard primary of db shop)"
EXPECT[mongo_collscans]="QAN (profiler): find/count on shop.orders by customer_id, status+amount and an unanchored note regex examine ~1M docs per call (COLLSCAN); mongod.log 'Slow query' entries with planSummary COLLSCAN on the primary of the shard that owns db shop; CPU and read tickets in use rise. Fix: createIndex({customer_id: 1}) and ({status: 1, amount: 1}); avoid unanchored regex."
start_mongo_collscans() {
  mjs "$MONGOS" mongo_collscans "${2:-6}" "$(app_uri shop_app)" "$1" <<'JS'
const o = db.getSiblingDB('shop').orders;
while (Date.now() < END) {
  const r = Math.floor(Math.random() * 3);
  if (r === 0) o.find({customer_id: 1 + Math.floor(Math.random() * 200000)}).toArray();
  else if (r === 1) o.countDocuments({status: 'paid', amount: {$gt: Math.random() * 500}});
  else o.find({note: new RegExp(Math.floor(Math.random() * 65536).toString(16).padStart(4, '0'))}).limit(20).toArray();
}
JS
}
stop_mongo_collscans() { stop_jobs "$MONGOS" mongo_collscans; }

TARGET[mongo_big_sorts]="MongoDB shop.orders via $MONGOS"
EXPECT[mongo_big_sorts]="Errors 'Sort exceeded memory limit of 104857600 bytes' (QueryExceededMemoryLimitNoDiskUseAllowed, code 292) for find on shop.orders sorted by note with allowDiskUse false: the shard primary's mongod.log has 'Plan executor error during find command' and 'Slow query' entries with that errName, mongos.log 'Unable to establish remote cursors'. QAN lists the find (COLLSCAN + SORT) but its error count stays 0. Fix: index {note: 1}, add a filter/limit, or allow disk use."
start_mongo_big_sorts() {
  mjs "$MONGOS" mongo_big_sorts 3 "$(app_uri shop_app)" "$1" <<'JS'
const o = db.getSiblingDB('shop').orders;
while (Date.now() < END) {
  try { o.find({}).sort({note: 1}).allowDiskUse(false).limit(0).batchSize(10).next(); } catch (e) {}
  sleep(500);
}
JS
}
stop_mongo_big_sorts() { stop_jobs "$MONGOS" mongo_big_sorts; }

TARGET[mongo_timeouts]="MongoDB shop.orders via $MONGOS"
EXPECT[mongo_timeouts]="Errors 'operation exceeded time limit' (MaxTimeMSExpired, code 50) for find by customer_id with maxTimeMS 30 and an aggregate with maxTimeMS 200 on shop.orders; mongod.log Slow query entries with errName MaxTimeMSExpired (in the log attributes); QAN error counts may stay 0; the queries are COLLSCANs needing ~1 s. Fix: index customer_id (and status) so they fit the time budget."
start_mongo_timeouts() {
  mjs "$MONGOS" mongo_timeouts 4 "$(app_uri shop_app)" "$1" <<'JS'
const o = db.getSiblingDB('shop').orders;
while (Date.now() < END) {
  try { o.find({customer_id: 1 + Math.floor(Math.random() * 200000)}).maxTimeMS(30).toArray(); } catch (e) {}
  try { o.aggregate([{$match: {status: 'paid'}}, {$group: {_id: '$customer_id', total: {$sum: '$amount'}}}], {maxTimeMS: 200}).toArray(); } catch (e) {}
  sleep(200);
}
JS
}
stop_mongo_timeouts() { stop_jobs "$MONGOS" mongo_timeouts; }

TARGET[mongo_write_storm]="MongoDB shop.orders / shop.audit_log via $MONGOS"
EXPECT[mongo_write_storm]="Write load on the shard primary owning db shop: updateMany on shop.orders by customer_id (COLLSCAN, ~1M docs examined per update) and bulk inserts into the capped shop.audit_log; write tickets in use, WiredTiger dirty cache and oplog churn up, CPU up, possible secondary lag. Fix: index customer_id; batch audit writes."
start_mongo_write_storm() {
  mjs "$MONGOS" mongo_write_storm 6 "$(app_uri shop_app)" "$1" <<'JS'
const s = db.getSiblingDB('shop');
while (Date.now() < END) {
  if (Math.random() < 0.5) {
    s.orders.updateMany({customer_id: 1 + Math.floor(Math.random() * 200000)}, {$inc: {amount: 1}, $set: {status: 'paid'}});
  } else {
    const docs = [];
    for (let i = 0; i < 500; i++) docs.push({at: new Date(), order_no: Math.floor(Math.random() * 1000000), event: 'recalc', payload: 'x'.repeat(400)});
    s.audit_log.insertMany(docs, {ordered: false});
  }
}
JS
}
stop_mongo_write_storm() { stop_jobs "$MONGOS" mongo_write_storm; }

TARGET[mongo_auth_failures]="MongoDB $MONGOS and rs101 (logs only)"
EXPECT[mongo_auth_failures]="Logs only: mongos.log and rs101 mongod.log 'Failed to authenticate' (id 5286307; attr.error AuthenticationFailed for shop_app@shop, UserNotFound for report_ro@admin), about 2 per second each from 127.0.0.1; no alert. Unrelated background: user pmm logs MechanismUnavailable for SCRAM-SHA-256 now and then. Cause: a client with a wrong password or brute force."
start_mongo_auth_failures() {
  local js
  js='while (Date.now() < END) {
  for (const u of ["mongodb://shop_app:winter2024@127.0.0.1:27017/shop?authSource=shop", "mongodb://report_ro:report@127.0.0.1:27017/admin"]) {
    try { new Mongo(u).getDB("admin").runCommand({ping: 1}); } catch (e) {}
  }
  sleep(1000);
}'
  echo "$js" | mjs "$MONGOS" mongo_auth_failures 1 "$(app_uri shop_web)" "$1"
  echo "$js" | mjs rs101 mongo_auth_failures 1 "mongodb://$MONGO_ROOT@127.0.0.1:27017/admin" "$1"
}
stop_mongo_auth_failures() {
  stop_jobs "$MONGOS" mongo_auth_failures
  stop_jobs rs101 mongo_auth_failures
}

TARGET[mongo_stepdown]="MongoDB replica set that owns db shop"
EXPECT[mongo_stepdown]="Alert 'MongoDB primary changed' for the shard replica set owning db shop; the old primary's mongod.log: stepDown command, 'Stepping down from primary', then an election on another member ('Starting an election', 'Election succeeded'); mongos.log replica set monitor changes; brief NotWritablePrimary errors for writers. Cause: a manual rs.stepDown (planned maintenance)."
start_mongo_stepdown() {
  local shard prim
  shard=$(mongo_shop_shard)
  prim=$(mongo_primary "$shard")
  log "stepping down $prim (primary of $shard)"
  mg "$prim" --eval 'try { rs.stepDown(120); } catch (e) { print(e.message); }' </dev/null >/dev/null
}
stop_mongo_stepdown() { :; }

TARGET[host_cpu_saturation]="Host (all engines at once)"
EXPECT[host_cpu_saturation]="Alert 'Host CPU high' (>70% of 16 cores): MySQL, PostgreSQL and MongoDB all run the unindexed scans of mysql_full_scans, pg_seq_scans and mongo_collscans at the same time; per-service QAN load shows which engine uses most; all engines' latency rises. Fix: the missing indexes on each engine."
start_host_cpu_saturation() {
  start_mysql_full_scans "$1" 12
  start_pg_seq_scans "$1" 12
  start_mongo_collscans "$1" 10
}
stop_host_cpu_saturation() {
  stop_mysql_full_scans
  stop_pg_seq_scans
  stop_mongo_collscans
}

### Runner

CURRENT=""
CURRENT_START=""

on_signal() {
  if [[ -n $CURRENT ]]; then
    log "interrupted: stopping $CURRENT"
    "stop_$CURRENT"
    truth "$CURRENT" "$CURRENT_START" "$(now)" "${TARGET[$CURRENT]}" "(interrupted) ${EXPECT[$CURRENT]}"
  fi
  rm -f "$STATE_DIR/current"
  exit 0
}

run_scenario() {
  local name=$1 secs=$2
  if ! declare -F "start_$name" >/dev/null; then
    echo "unknown scenario: $name (see: $0 list)" >&2
    return 1
  fi
  CURRENT=$name
  CURRENT_START=$(now)
  echo "$name $CURRENT_START $secs" >"$STATE_DIR/current"
  log "start $name for ${secs}s"
  "start_$name" "$secs"
  sleep "$secs" &
  wait $!
  "stop_$name"
  truth "$name" "$CURRENT_START" "$(now)" "${TARGET[$name]}" "${EXPECT[$name]}"
  log "end $name"
  CURRENT=""
  rm -f "$STATE_DIR/current"
}

chaos_loop() {
  local dur=$1 gap=$2
  shift 2
  local list=("$@")
  [[ ${#list[@]} -gt 0 ]] || list=("${ROTATION[@]}")
  while true; do
    for s in "${list[@]}"; do
      run_scenario "$s" "$dur"
      sleep "$gap" &
      wait $!
    done
  done
}

cmd_chaos_start() {
  local dur=${1:-10} gap=${2:-10}
  shift 2 2>/dev/null || shift $#
  if [[ -f $STATE_DIR/chaos.pid ]] && kill -0 "$(cat "$STATE_DIR/chaos.pid")" 2>/dev/null; then
    echo "chaos loop already running (pid $(cat "$STATE_DIR/chaos.pid"))"
    return 0
  fi
  setsid nohup "$0" _loop "$((dur * 60))" "$((gap * 60))" "$@" >>"$STATE_DIR/chaos.log" 2>&1 </dev/null &
  echo $! >"$STATE_DIR/chaos.pid"
  log "chaos loop started (pid $!): ${dur} min incidents, ${gap} min calm; log $STATE_DIR/chaos.log, ground truth $TRUTH"
}

cmd_chaos_stop() {
  if [[ -f $STATE_DIR/chaos.pid ]]; then
    local pid
    pid=$(cat "$STATE_DIR/chaos.pid")
    kill -TERM "$pid" 2>/dev/null
    for _ in $(seq 60); do
      kill -0 "$pid" 2>/dev/null || break
      sleep 1
    done
    rm -f "$STATE_DIR/chaos.pid"
    log "chaos loop stopped"
  fi
}

cmd_stop() {
  cmd_chaos_stop
  cmd_baseline_stop
  local c
  for c in "$MYSQL_PRIMARY" "$PG" "$MONGOS" rs101; do stop_jobs "$c" ""; done
  kill_mysql_app
  kill_pg_app
  log "all workloads stopped"
}

cmd_status() {
  if [[ -f $STATE_DIR/chaos.pid ]] && kill -0 "$(cat "$STATE_DIR/chaos.pid")" 2>/dev/null; then
    echo "chaos loop: running (pid $(cat "$STATE_DIR/chaos.pid"))"
  else
    echo "chaos loop: stopped"
  fi
  [[ -f $STATE_DIR/current ]] && echo "current incident: $(cat "$STATE_DIR/current")"
  local c
  for c in "$MYSQL_PRIMARY" "$PG" "$MONGOS" rs101; do echo "jobs in $c: $(running_jobs "$c")"; done
  echo "last incidents ($TRUTH):"
  [[ -f $TRUTH ]] && tail -n "${1:-10}" "$TRUTH" | python3 -c 'import json, sys
for l in sys.stdin:
    d = json.loads(l)
    print("  %s .. %s  %-24s %s" % (d["start"], d["end"], d["scenario"], d["target"]))'
}

cmd_list() {
  local s
  for s in "${ROTATION[@]}"; do
    printf '%s\n  target: %s\n  expect: %s\n\n' "$s" "${TARGET[$s]}" "${EXPECT[$s]}"
  done
}

usage() {
  cat <<EOF
usage: $0 <command>
  setup                         create users, data and log settings (idempotent)
  baseline start|stop           steady shop_web traffic on all three engines
  run <scenario> [minutes]      run one incident in the foreground (default 10 minutes)
  chaos start [min] [gap] [s..] rotate incidents in the background (default 10 / 10 minutes, all scenarios)
  chaos stop                    stop the rotation, undoing the current incident
  status [n]                    loop state, running workloads, last n incidents
  list                          scenarios with target and expected evidence
  stop                          stop everything (rotation, baseline, incidents)
  teardown                      stop, drop the shop data and restore settings
EOF
}

trap on_signal TERM INT

case "${1:-}" in
  setup) cmd_setup ;;
  teardown) cmd_teardown ;;
  baseline) case "${2:-}" in start) cmd_baseline_start ;; stop) cmd_baseline_stop ;; *) usage ;; esac ;;
  run) run_scenario "${2:?scenario}" "$((${3:-10} * 60))" ;;
  chaos) case "${2:-}" in start) shift 2; cmd_chaos_start "$@" ;; stop) cmd_chaos_stop ;; *) usage ;; esac ;;
  _loop) shift; chaos_loop "$@" ;;
  status) cmd_status "${2:-10}" ;;
  list) cmd_list ;;
  stop) cmd_stop ;;
  *) usage ;;
esac
