# pmm-managed Development Guidelines

**pmm-managed** manages the configuration of [PMM](https://docs.percona.com/percona-monitoring-and-management/3/) server components (VictoriaMetrics, Grafana, QAN, etc.) and exposes an API for interacting with them. The API is also consumed by [pmm-admin tool](https://github.com/percona/pmm/tree/main/admin).

## Architecture Patterns

### Database Layer (reform ORM)

PMM uses **reform** (NOT gorm) for PostgreSQL interactions:

```go
// Models are defined with reform tags and generated code
//go:generate ../../bin/reform

//reform:nodes
type Node struct {
    NodeID   string   `reform:"node_id,pk"`
    NodeName string   `reform:"node_name"`
    // ...
}

// All DB operations use reform.Querier
func FindNodeByID(q *reform.Querier, id string) (*Node, error) {
    // Use reform methods, not raw SQL when possible
}

// Transactions use reform.TX
db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
    // Transaction logic
})
```

**Key points:**
- Models live in `managed/models/`
- `*_model.go` files have `//go:generate` directives
- `*_helpers.go` files contain CRUD operations
- Always use `reform.Querier` parameter, not concrete types
- Check for `reform.ErrNoRows` explicitly

### Service Architecture

Services follow a consistent pattern in `managed/services/`:

```go
type Service struct {
    db       *reform.DB
    l        *logrus.Entry
    // dependencies
}

// Constructor with dependency injection
func New(db *reform.DB, logger *logrus.Entry) *Service {
    return &Service{db: db, l: logger}
}
```

Services are composed in `managed/cmd/pmm-managed/main.go` and injected throughout the application.

### API Definitions (Protocol Buffers)

APIs are defined in `.proto` files under `/api/`:
- Generate with: `make gen` from the root of the repository
- Creates Go code, Swagger specs, and gRPC gateway mappings
- **Never** edit generated files (`.pb.go`, `.pb.gw.go`, swagger files)
- Update proto files, then regenerate

### High Availability (HA)

PMM supports HA using **Raft consensus** (`/managed/services/ha/`):
- Distributed state is managed via Raft
- pmm-agent states are synchronized across nodes
- Uses `hashicorp/raft` library
- Critical for ensuring consistency in multi-node setups

## Testing Conventions

### Unit Tests
- Use `testify/assert` and `testify/require`
- Mock generation via `mockery` (config in `.mockery.yaml`)
- Use `testdb` helper for DB tests

### Integration Tests
- Located in `/api-tests/` (separate from unit tests)
- Use `testify/assert` and `testify/require`
- Setup/teardown pattern with `testdb.Open()` helper
- Run against live PMM Server: `make api-test`

## Code Generation

Multiple code generation tools are used:

1. **Protocol Buffers** - APIs (`make gen` from root)
2. **reform** - ORM model generation (`//go:generate ../../bin/reform`)
3. **mockery** - Mock generation for interfaces
4. **swagger** - API documentation

**Always run `make gen` after:**
- Adding/modifying `.proto` files
- Adding/modifying reform models
- Changing interface signatures that need mocks

## Common Patterns

### Do
- Prefer modern Go idioms (context, error wrapping)
- Prefer modern slice helpers (e.g., `slices.Contains`), range loops
- Use `any` instead of `interface{}`

### Don't
- Don't use `gorm` or other ORMs - only `reform`
- Don't edit generated files manually
- Don't create subshells in Makefiles without explicit reason
- Don't skip `make gen` after proto/model changes
- Don't commit test binaries or test artifacts (add to `.gitignore` if needed)
- Don't comment on every single line of code unnecessarily, only where clarity is needed
- Don't inline comments (i.e. `code // comment`), always put comments on separate lines

### Error Handling
- Use `status.Error()` for gRPC errors with proper codes
- Check `reform.ErrNoRows` for "not found" scenarios
- Wrap errors with context: `fmt.Errorf("descriptive context: %w", err)`
- Return early on errors to avoid deep nesting
- Use `errors.Is()` and `errors.As()` for error type checking
- Use `errors.WithStack()` wisely and only when stack traces are needed
- Use standard `errors` package instead of `github.com/pkg/errors`

### Logging
- Use structured logging with `logrus`
- Pass `*logrus.Entry` (not `*logrus.Logger`) to maintain context
- Format: `s.l.WithField("key", value).Error("message")`

## Agent Management
- Agents are registered and managed via `managed/services/agents/registry.go`
- Communication uses bidirectional gRPC streams
- Agent states are tracked in PostgreSQL and synchronized with HA state machine

## RESTful conventions
- Use RESTful conventions (GET/POST/PUT/DELETE with resource paths)
- Use custom endpoints only when necessary (e.g., actions)

## Key Files to Reference

- `managed/models/database.go` - Database schema and migrations
- `managed/cmd/pmm-managed/main.go` - Application bootstrap and wiring
- `docker-compose.yml` - Development environment configuration
- `Makefile` and `Makefile.include` - Common make targets
- `.devcontainer/setup.py` - Devcontainer initialization
