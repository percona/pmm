# Database Load Generator

A comprehensive Go application for generating database load across multiple database types (MySQL, PostgreSQL, MongoDB) with unoptimized queries that demonstrate common developer mistakes.

## Features

- **Multi-Database Support**: MySQL, PostgreSQL, and MongoDB
- **Comprehensive Unoptimized Queries**: 60+ different anti-patterns and common mistakes
- **Modular Architecture**: Service-based design for easy extension
- **Embedded Migrations**: Database-specific migrations built into the binary
- **Automatic Data Seeding**: Pre-populates databases with realistic test data
- **Self-Contained Binary**: All migrations and schema embedded, no external files needed
- **Graceful Shutdown**: Handles interrupts cleanly
- **Real-time Metrics**: Live operation counters per worker
- **Docker Support**: Complete development environment

## Data Seeding

The application automatically seeds databases with substantial test data to make anti-patterns meaningful:

### MySQL & PostgreSQL Tables
- **1,000 users** with profiles, birth dates, and JSON/JSONB data
- **2,000 orders** with user relationships and order details
- **5,000 order items** linked to orders for complex JOIN scenarios
- **50 categories** with hierarchical parent-child relationships
- **10,000 audit log entries** for large table scan demonstrations

### MongoDB Collections
- **1,000 users** with nested objects, arrays, and varied document structures
- **2,000 orders** with embedded item arrays and shipping addresses
- **500 products** with categories, ratings, and searchable descriptions
- **10,000 audit log entries** for large collection scan scenarios
- **5,000 events** with time-series data and string date formats
- **1,000 locations** with GeoJSON coordinates for spatial queries

This data is inserted only once (skipped if >100 users already exist) and provides the foundation for realistic performance testing of unoptimized queries.

## Anti-Patterns Demonstrated

### MySQL & PostgreSQL
- **SELECT * queries** - Retrieving all columns instead of specific fields
- **N+1 queries** - Classic problem causing excessive database round trips
- **Missing WHERE clauses** - Full table scans on large datasets
- **Functions in WHERE clauses** - Preventing index usage with YEAR(), UPPER(), etc.
- **Leading wildcard LIKE** - Searches like '%pattern' that can't use indexes
- **Inefficient subqueries** - Using subqueries instead of JOINs
- **Missing LIMIT clauses** - Unbounded result sets
- **Unnecessary DISTINCT** - Using DISTINCT on already unique data
- **Unindexed ORDER BY** - Sorting on non-indexed columns
- **Complex CASE statements** - Nested CASE logic in SELECT clauses
- **Inefficient OR conditions** - Multiple ORs instead of optimized approaches
- **Unoptimized aggregations** - GROUP BY without supporting indexes
- **Inefficient self-joins** - Deep hierarchical queries without proper indexing
- **JSON/JSONB operations** - Without proper GIN/functional indexes
- **Correlated subqueries** - Subqueries executing for each row
- **Multiple table scans** - Multiple full scans in single query
- **Non-sargable date operations** - Date functions preventing index usage
- **Large IN clauses** - Very large IN operations (1000+ values)
- **Inefficient GROUP BY** - Grouping with functions and no supporting indexes

### PostgreSQL-Specific
- **Inefficient window functions** - Without proper partitioning
- **Recursive CTEs** - Without proper termination conditions
- **JSONB operations** - Complex operations without GIN indexes
- **Array operations** - Inefficient array queries

### MongoDB-Specific
- **Large collection scans** - No indexes with complex filters
- **N+1 aggregation** - Multiple queries instead of single aggregation
- **Inefficient regex** - Leading wildcards and case-insensitive searches
- **Missing compound indexes** - Queries requiring multiple field indexes
- **Large skip() operations** - Inefficient pagination
- **Over-fetching** - Retrieving entire documents when only fields needed
- **Inefficient array queries** - Array operations without proper indexing
- **Poor aggregation pipelines** - $match late in pipeline, unnecessary $unwind
- **Wrong data types** - String vs ObjectId, mismatched types
- **Inefficient text search** - Regex instead of text indexes
- **Unoptimized geo queries** - Without 2dsphere indexes
- **Memory-intensive operations** - Operations exceeding memory limits
- **Inefficient date ranges** - String comparisons instead of Date objects
- **Large $in operations** - Very large $in arrays
- **Inefficient counting** - find().count() instead of countDocuments
- **Multiple round trips** - Separate queries instead of aggregation

## Installation

```bash
# Clone or copy the db-loadgen directory
cd db-loadgen

# Install dependencies
go mod download

# Or run make deps
make deps
```

## Usage

### Command Line Flags

- `--mysql-dsn`: MySQL connection string (optional)
- `--postgres-dsn`: PostgreSQL connection string (optional)
- `--mongo-dsn`: MongoDB connection string (optional)
- `--duration`: Test duration (e.g., 60s, 5m, 1h). If not specified, runs indefinitely
- `--workers`: Number of workers per database (default: 5)

### Running with Docker Compose

```bash
# Start test databases
docker compose up -d

# Wait for databases to be ready (30 seconds)
sleep 30

# Run load generator against all databases
make run-all

# Or run against specific databases
make run-mysql
make run-postgres
make run-mongo
```

### Manual Usage

```bash
# Build the application
make build

# Run with custom settings
./db-loadgen \
  --mysql-dsn="user:pass@tcp(localhost:3306)/testdb" \
  --postgres-dsn="postgresql://user:pass@localhost:5432/testdb?sslmode=disable" \
  --mongo-dsn="mongodb://user:pass@localhost:27017/testdb" \
  --duration=60s \
  --workers=10
```

## Database Schema

The application creates comprehensive table structures designed to demonstrate various anti-patterns:

### MySQL/PostgreSQL Tables
- **users** - User profiles with JSON/JSONB data, minimal indexing
- **orders** - Order records with foreign keys but missing indexes
- **order_items** - Order line items for JOIN scenarios
- **audit_log** - Large logging table without indexes (except primary)
- **categories** - Hierarchical data for self-join scenarios

### MongoDB Collections
- **users** - User documents with nested objects and arrays
- **orders** - Order documents with embedded items
- **products** - Product catalog for text search scenarios
- **audit_log** - Event logging collection
- **events** - User activity tracking
- **locations** - Geospatial data for location queries

## Performance Impact

Each anti-pattern query is designed to demonstrate real performance issues:

- **CPU Usage**: Complex calculations, regex operations, function calls
- **Memory Usage**: Large result sets, unnecessary data fetching
- **I/O Load**: Full table scans, inefficient index usage
- **Network Traffic**: Over-fetching data, N+1 queries
- **Lock Contention**: Long-running queries, table scans

## Educational Value

This tool is perfect for:

- **Performance Testing**: Identifying bottlenecks and resource limits
- **Query Optimization Training**: Learning what NOT to do
- **Database Monitoring**: Testing monitoring tools with realistic bad queries
- **Index Strategy**: Understanding index design importance
- **Code Review**: Recognizing anti-patterns in applications

## Development

### Project Structure

```
db-loadgen/
├── main.go                    # Main application entry point
├── service/                   # Database-specific service packages
│   ├── mysql/mysql.go        # MySQL unoptimized operations
│   ├── postgres/postgres.go  # PostgreSQL unoptimized operations
│   └── mongo/mongo.go        # MongoDB unoptimized operations
├── migrations/               # Database migrations (embedded into binary)
│   ├── embedded.go          # Embedding logic for migration files
│   ├── mysql/               # MySQL schema files
│   └── postgres/           # PostgreSQL schema files
├── go.mod                   # Go dependencies
├── Makefile                # Build and run targets
├── docker-compose.yml      # Test database environment
└── README.md              # This file
```

### Embedded Migrations

The application uses Go's `embed` package to include migration files directly in the binary:

- **No external files needed**: All SQL migrations are embedded at compile time
- **Self-contained deployment**: Single binary contains everything needed
- **Version consistency**: Migration files always match the binary version
- **Easy distribution**: No risk of missing migration files

The embedding is handled by `migrations/embedded.go` which makes migration files available through the `io/fs` interface.

### Adding New Anti-Patterns

1. Add new method to appropriate service (mysql.go, postgres.go, mongo.go)
2. Update the switch statement in `performUnoptimizedOperation`
3. Document the anti-pattern in this README
4. Test with the included database environment

### Building

```bash
# Install dependencies
make deps

# Build binary
make build

# Clean build artifacts
make clean

# Format code
go fmt ./...

# Run linter
golangci-lint run
```

## Environment Variables

Create `.env` file for custom settings:

```env
MYSQL_DSN=user:password@tcp(localhost:3306)/testdb
POSTGRES_DSN=postgresql://user:password@localhost:5432/testdb?sslmode=disable
MONGO_DSN=mongodb://user:password@localhost:27017/testdb
DURATION=30s
WORKERS=5
```

## Troubleshooting

### Connection Issues
- Ensure databases are running: `docker-compose ps`
- Check connection strings format
- Verify database credentials

### Migration Failures
- Check database connectivity
- Ensure proper permissions for DDL operations
- Migrations are embedded in the binary - no external files needed
- If migrations fail, the issue is likely database permissions or connectivity

### Data Seeding Issues
- Seeding runs automatically after migrations and before load generation
- If interrupted, restart the application to continue from where it left off
- Seeding is skipped if >100 users already exist in the database
- For complete reset, drop/recreate databases before running

### Performance Issues
- Reduce worker count if system overloaded
- Monitor system resources during load generation
- Check database slow query logs
- Initial seeding may take 30-60 seconds depending on system performance

## License

This project is for educational and testing purposes. Use responsibly and never run these anti-patterns in production environments! 