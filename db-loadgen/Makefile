# Default DSN values for docker-compose environment
DEFAULT_MYSQL_DSN=testuser:testpass@tcp(localhost:3306)/testdb?multiStatements=true
DEFAULT_POSTGRES_DSN=postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable
DEFAULT_MONGO_DSN=mongodb://testuser:testpass@localhost:27017/loadtest?authSource=loadtest

# PMM configuration
PMM_ADMIN_BINARY?=pmm-admin
PMM_SERVER_URL=https://localhost:443
PMM_SERVER_USER=admin
PMM_SERVER_PASSWORD=admin

.PHONY: build clean run-mysql run-postgres run-mongo run-all run-mysql-infinite run-postgres-infinite run-mongo-infinite run-all-infinite test deps help pmm-add-mysql pmm-add-postgres pmm-add-mongo pmm-add-all pmm-remove-mysql pmm-remove-postgres pmm-remove-mongo pmm-remove-all pmm-status pmm-check

# Binary name
BINARY_NAME=db-loadgen

# Default target
help:
	@echo "Available targets:"
	@echo "  build        - Build the application"
	@echo "  clean        - Clean build artifacts"
	@echo "  deps         - Download dependencies"
	@echo "  test         - Run tests"
	@echo "  run-mysql    - Run with MySQL (requires MYSQL_DSN)"
	@echo "  run-postgres - Run with PostgreSQL (requires POSTGRES_DSN)"
	@echo "  run-mongo    - Run with MongoDB (requires MONGO_DSN)"
	@echo "  run-all      - Run with all databases in single process"
	@echo ""
	@echo "Infinite Duration (Ctrl+C to stop):"
	@echo "  run-mysql-infinite    - Run MySQL load indefinitely"
	@echo "  run-postgres-infinite - Run PostgreSQL load indefinitely"
	@echo "  run-mongo-infinite    - Run MongoDB load indefinitely"
	@echo "  run-all-infinite      - Run all databases indefinitely"
	@echo ""
	@echo "PMM Integration:"
	@echo "  pmm-check        - Check if PMM admin is available"
	@echo "  pmm-add-mysql    - Add MySQL service to PMM monitoring"
	@echo "  pmm-add-postgres - Add PostgreSQL service to PMM monitoring"
	@echo "  pmm-add-mongo    - Add MongoDB service to PMM monitoring"
	@echo "  pmm-add-all      - Add all database services to PMM monitoring"
	@echo "  pmm-remove-mysql    - Remove MySQL service from PMM monitoring"
	@echo "  pmm-remove-postgres - Remove PostgreSQL service from PMM monitoring"
	@echo "  pmm-remove-mongo    - Remove MongoDB service from PMM monitoring"
	@echo "  pmm-remove-all      - Remove all database services from PMM monitoring"
	@echo "  pmm-status       - Show PMM monitoring status"
	@echo ""
	@echo "Configuration:"
	@echo "  PMM_ADMIN_BINARY - Path to pmm-admin binary (default: pmm-admin)"
	@echo "  Example: make pmm-add-all PMM_ADMIN_BINARY=/usr/local/bin/pmm-admin"

# Build the application
build:
	go build -o $(BINARY_NAME) main.go

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run tests (placeholder for when tests are added)
test:
	go test -v ./...

# Run with MySQL only
run-mysql: build
	@if [ -z "$(MYSQL_DSN)" ]; then \
	  export MYSQL_DSN='$(DEFAULT_MYSQL_DSN)'; \
	fi; \
	./$(BINARY_NAME) --mysql-dsn "$$MYSQL_DSN" --duration 1m --workers 5

# Run with PostgreSQL only
run-postgres: build
	@if [ -z "$(POSTGRES_DSN)" ]; then \
	  export POSTGRES_DSN='$(DEFAULT_POSTGRES_DSN)'; \
	fi; \
	./$(BINARY_NAME) --postgres-dsn "$$POSTGRES_DSN" --duration 1m --workers 5

# Run with MongoDB only
run-mongo: build
	@if [ -z "$(MONGO_DSN)" ]; then \
	  export MONGO_DSN='$(DEFAULT_MONGO_DSN)'; \
	fi; \
	./$(BINARY_NAME) --mongo-dsn "$$MONGO_DSN" --duration 1m --workers 5

# Run with all databases
run-all: build
	@if [ -z "$(MYSQL_DSN)" ]; then \
	  export MYSQL_DSN='$(DEFAULT_MYSQL_DSN)'; \
	fi; \
	if [ -z "$(POSTGRES_DSN)" ]; then \
	  export POSTGRES_DSN='$(DEFAULT_POSTGRES_DSN)'; \
	fi; \
	if [ -z "$(MONGO_DSN)" ]; then \
	  export MONGO_DSN='$(DEFAULT_MONGO_DSN)'; \
	fi; \
	./$(BINARY_NAME) --mysql-dsn "$$MYSQL_DSN" --postgres-dsn "$$POSTGRES_DSN" --mongo-dsn "$$MONGO_DSN" --duration 2m --workers 10

# Infinite duration targets (run until interrupted)
run-mysql-infinite: build
	@if [ -z "$(MYSQL_DSN)" ]; then \
	  export MYSQL_DSN='$(DEFAULT_MYSQL_DSN)'; \
	fi; \
	echo "Starting MySQL load generation indefinitely. Press Ctrl+C to stop."; \
	./$(BINARY_NAME) --mysql-dsn "$$MYSQL_DSN" --workers 5

run-postgres-infinite: build
	@if [ -z "$(POSTGRES_DSN)" ]; then \
	  export POSTGRES_DSN='$(DEFAULT_POSTGRES_DSN)'; \
	fi; \
	echo "Starting PostgreSQL load generation indefinitely. Press Ctrl+C to stop."; \
	./$(BINARY_NAME) --postgres-dsn "$$POSTGRES_DSN" --workers 5

run-mongo-infinite: build
	@if [ -z "$(MONGO_DSN)" ]; then \
	  export MONGO_DSN='$(DEFAULT_MONGO_DSN)'; \
	fi; \
	echo "Starting MongoDB load generation indefinitely. Press Ctrl+C to stop."; \
	./$(BINARY_NAME) --mongo-dsn "$$MONGO_DSN" --workers 5

run-all-infinite: build
	@if [ -z "$(MYSQL_DSN)" ]; then \
	  export MYSQL_DSN='$(DEFAULT_MYSQL_DSN)'; \
	fi; \
	if [ -z "$(POSTGRES_DSN)" ]; then \
	  export POSTGRES_DSN='$(DEFAULT_POSTGRES_DSN)'; \
	fi; \
	if [ -z "$(MONGO_DSN)" ]; then \
	  export MONGO_DSN='$(DEFAULT_MONGO_DSN)'; \
	fi; \
	echo "Starting load generation for all databases indefinitely. Press Ctrl+C to stop."; \
	./$(BINARY_NAME) --mysql-dsn "$$MYSQL_DSN" --postgres-dsn "$$POSTGRES_DSN" --mongo-dsn "$$MONGO_DSN" --workers 10

# Install target (build and move to PATH)
install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Development setup
dev-setup: deps build
	@echo "Development setup complete. Run 'make help' for usage."

# PMM Integration targets
pmm-check:
	@echo "Checking PMM admin availability..."
	@which $(PMM_ADMIN_BINARY) > /dev/null || (echo "Error: $(PMM_ADMIN_BINARY) not found in PATH. Please install PMM client or set PMM_ADMIN_BINARY." && exit 1)
	@echo "PMM admin found: $$(which $(PMM_ADMIN_BINARY))"
	@echo "PMM admin version: $$($(PMM_ADMIN_BINARY) --version)"

pmm-add-mysql: pmm-check
	@echo "Adding MySQL service to PMM monitoring..."
	$(PMM_ADMIN_BINARY) add mysql \
		--username=pmm-mysql \
		--password=pmm-pass \
		--host=localhost \
		--port=3306 \
		--service-name=loadgen-mysql \
		--environment=development \
		--cluster=loadgen-cluster \
		--query-source=perfschema

pmm-add-postgres:
	@echo "Adding PostgreSQL service to PMM monitoring..."
	$(PMM_ADMIN_BINARY) add postgresql \
		--username=pmm-postgres \
		--password=pmm-pass \
		--host=localhost \
		--port=5432 \
		--database=testdb \
		--service-name=loadgen-postgres \
		--environment=development \
		--cluster=loadgen-cluster

pmm-add-mongo:
	@echo "Adding MongoDB service to PMM monitoring..."
	$(PMM_ADMIN_BINARY) add mongodb \
		--username=pmm-mongodb \
		--password=pmm-pass \
		--host=localhost \
		--port=27017 \
		--service-name=loadgen-mongodb \
		--environment=development \
		--cluster=loadgen-cluster \
		--authentication-database=admin

pmm-add-all: pmm-add-mysql pmm-add-postgres pmm-add-mongo
	@echo "All database services added to PMM monitoring"

pmm-remove-mysql:
	@echo "Removing MySQL service from PMM monitoring..."
	-$(PMM_ADMIN_BINARY) remove mysql loadgen-mysql

pmm-remove-postgres:
	@echo "Removing PostgreSQL service from PMM monitoring..."
	-$(PMM_ADMIN_BINARY) remove postgresql loadgen-postgres

pmm-remove-mongo:
	@echo "Removing MongoDB service from PMM monitoring..."
	-$(PMM_ADMIN_BINARY) remove mongodb loadgen-mongodb

pmm-remove-all: pmm-remove-mysql pmm-remove-postgres pmm-remove-mongo
	@echo "All database services removed from PMM monitoring"

pmm-status:
	@echo "PMM monitoring status:"
	$(PMM_ADMIN_BINARY) status
	@echo ""
	@echo "PMM services list:"
	$(PMM_ADMIN_BINARY) list 