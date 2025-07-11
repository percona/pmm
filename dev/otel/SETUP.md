# OpenTelemetry PoC Setup Instructions

## Project Structure
This PoC has the following directory structure:

```
otel/
├── docker-compose.yml
├── config.yml
├──clickhouse/
│   ├── init.sql
│   └── test.sql
├── doc/
├── nginx/
│   └── nginx.conf
└── vector/
    ├── Dockerfile
    └── vector.toml

```

## Setup Steps

### 1. Clone the Project
```bash
git clone https://github.com/percona/pmm.git
cd pmm/dev/otel
```

### 4. Start the Environment
```bash
# Start all services
docker compose up -d

# Check service status
docker compose ps

# View logs
docker compose logs -f otel-collector
docker compose logs -f pmm-server
```

### 5. Generate Logs
PMM is indeed generating quite some logs, but you can also generate logs for testing purposes:

```bash
# Generate various HTTP responses
curl -k -u admin:admin https://localhost/                    # 200 OK
curl -k -u admin:admin https://localhost/api/users           # 200 OK
curl -k -u admin:admin https://localhost/api/products        # 201 Created
curl -k -u admin:admin https://localhost/admin               # 403 Forbidden
curl -k -u admin:admin https://localhost/nonexistent         # 404 Not Found
curl -k -u admin:admin https://localhost/server-error        # 500 Internal Server Error
```

### 6. Access ClickHouse
```bash
# Connect to ClickHouse CLI
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel

# Or use HTTP interface
curl -k -u admin:admin "http://localhost:8123/?user=default&password=clickhouse&database=otel" -d "SELECT count() FROM logs"
```

### 7. Run Test Queries
Execute the test queries from the `clickhouse/test.sql` file in the ClickHouse client.

## Troubleshooting

### Check OpenTelemetry Collector Status
```bash
# View collector logs
docker compose logs otel-collector

# Check collector metrics
curl http://localhost:8888/metrics
```

### ClickHouse Data Verification
```bash
# Check table exists and has data
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel -q "SELECT count() FROM logs"

# View recent logs
docker exec -it pmm-server clickhouse-client --user=default --password=clickhouse --database=otel -q "SELECT * FROM logs ORDER BY timestamp DESC LIMIT 10"
```

## Services and Ports

- **PMM**: https://localhost:443
- **ClickHouse HTTP**: http://localhost:8123
- **ClickHouse Native**: localhost:9000
- **OpenTelemetry Collector Metrics**: http://localhost:8888/metrics
- **OpenTelemetry OTLP gRPC**: localhost:4317
- **OpenTelemetry OTLP HTTP**: localhost:4318

## Cleanup
```bash
# Stop and remove all containers
docker compose down

# Remove volumes (this will delete all data)
docker compose down -v

# Remove images
docker compose down --rmi all
```
