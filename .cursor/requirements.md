# Real-Time Query Analytics (RTA) Feature Requirements

## Overview
Real-Time Query Analytics (RTA) is a feature for Percona Monitoring and Management (PMM) that provides real-time visibility into currently running database queries, starting with MongoDB support.

## Core Architecture Requirements

### ✅ API & Protocol Design
- [x] **RTA-001**: Define protobuf messages for real-time analytics communication
- [x] **RTA-002**: Create gRPC service definition for agent-to-server communication
- [x] **RTA-003**: Design HTTP REST API endpoints for UI communication
- [x] **RTA-004**: Implement proper error handling and status codes
- [x] **RTA-005**: Support for both current and historical data retrieval

### ✅ Database Agent Implementation
- [x] **RTA-006**: MongoDB real-time analytics agent using `db.currentOp()` 
- [x] **RTA-007**: Collection of currently running queries with microsecond precision
- [x] **RTA-008**: Query fingerprinting using shared PMM fingerprinter
- [x] **RTA-009**: Configurable collection intervals (default: 1 second)
- [x] **RTA-010**: Query text extraction with privacy controls (disable examples)
- [x] **RTA-011**: Support for MongoDB 7.x `$currentOp` aggregation pipeline
- [x] **RTA-012**: Raw `currentOp` document preservation for debugging
- [x] **RTA-013**: Operation ID (`opid`) tracking for query deduplication
- [x] **RTA-014**: Panic recovery and error handling for robustness

### ✅ Backend Server Implementation
- [x] **RTA-015**: Real-time analytics gRPC service in pmm-managed
- [x] **RTA-016**: Data buffering with configurable retention (default: 2 minutes)
- [x] **RTA-017**: Service metadata enrichment from agent data
- [x] **RTA-018**: Label-Based Access Control (LBAC) integration
- [x] **RTA-019**: Historical vs. current data separation
- [x] **RTA-020**: Per-service data isolation and management

### ✅ Agent Management & Lifecycle
- [x] **RTA-021**: RTA agent type registration in inventory system
- [x] **RTA-022**: Agent configuration via `SetStateRequest` from pmm-managed
- [x] **RTA-023**: Automatic agent creation when adding MongoDB services
- [x] **RTA-024**: Agent status reporting and health monitoring
- [x] **RTA-025**: Connection management and error recovery
- [x] **RTA-026**: Graceful agent start/stop/restart capabilities

### ✅ Configuration Management
- [x] **RTA-027**: Store RTA configuration in JSONB field within `agent` table
- [x] **RTA-028**: Support for enable/disable per service
- [x] **RTA-029**: Configurable collection intervals
- [x] **RTA-030**: Query text privacy controls (disable examples)
- [x] **RTA-031**: Configuration validation and defaults

### ✅ Communication Infrastructure
- [x] **RTA-032**: pmm-agent to pmm-managed gRPC streaming
- [x] **RTA-033**: Efficient batch data transmission
- [x] **RTA-034**: Connection reuse and optimization
- [x] **RTA-035**: Authentication and authorization for gRPC endpoints
- [x] **RTA-036**: Nginx reverse proxy configuration for gRPC/HTTP routing

## User Interface Requirements

### ✅ Core UI Components
- [x] **RTA-037**: Real-time query analytics dashboard page
- [x] **RTA-038**: Service selection and filtering interface
- [x] **RTA-039**: Query table with sortable columns (fingerprint, database, duration, state, timestamp)
- [x] **RTA-040**: Query details dialog with raw MongoDB document display
- [x] **RTA-041**: Real-time data refresh with configurable intervals
- [x] **RTA-042**: Search and filtering capabilities
- [x] **RTA-043**: State-based query filtering (running, waiting, finished)

### ✅ Data Visualization
- [x] **RTA-044**: Query execution time display with millisecond precision
- [x] **RTA-045**: Query state indicators with color coding
- [x] **RTA-046**: Database and operation type labeling
- [x] **RTA-047**: Formatted JSON display for raw MongoDB operations
- [x] **RTA-048**: Service metadata display (node, labels)

### ✅ User Experience
- [x] **RTA-049**: Auto-refresh with 2-second intervals
- [x] **RTA-050**: Query deduplication by MongoDB `opid`
- [x] **RTA-051**: Latest query prioritization for duplicate operations
- [x] **RTA-052**: Responsive design with Material-UI components
- [x] **RTA-053**: Error handling for missing or undefined data
- [x] **RTA-054**: Loading states and empty state handling

## Security & Access Control Requirements

### ✅ Authentication & Authorization
- [x] **RTA-055**: LBAC integration following qan-api2 patterns
- [x] **RTA-056**: Service-level access control
- [x] **RTA-057**: Role-based permissions (viewer role minimum)
- [x] **RTA-058**: Secure gRPC communication with proper authentication
- [x] **RTA-059**: Query text privacy controls per service configuration

### ✅ Data Privacy
- [x] **RTA-060**: Optional query text collection (can be disabled)
- [x] **RTA-061**: Raw operation data sanitization
- [x] **RTA-062**: Service isolation and access controls
- [x] **RTA-063**: No sensitive credential exposure in logs or UI

## CLI Integration Requirements

### ✅ pmm-admin Integration
- [x] **RTA-064**: `pmm-admin add mongodb` command enables RTA by default
- [x] **RTA-065**: RTA configuration options in CLI
- [x] **RTA-066**: Agent type validation and compatibility checks
- [x] **RTA-067**: Proper error handling and user feedback

## Performance & Scalability Requirements

### ✅ Data Collection Efficiency
- [x] **RTA-068**: Minimal impact on monitored MongoDB instances
- [x] **RTA-069**: Configurable collection intervals to balance load
- [x] **RTA-070**: Efficient `$currentOp` aggregation with proper filtering
- [x] **RTA-071**: Connection pooling and reuse
- [x] **RTA-072**: Prometheus metrics for monitoring collection performance

### ✅ Data Management
- [x] **RTA-073**: Automatic data expiration (2-minute history buffer)
- [x] **RTA-074**: Memory-efficient data structures
- [x] **RTA-075**: Batch processing for efficient network usage
- [x] **RTA-076**: Separate current vs. historical data handling

### ✅ Network Optimization
- [x] **RTA-077**: gRPC streaming for real-time data
- [x] **RTA-078**: Connection reuse to minimize overhead
- [x] **RTA-079**: Efficient serialization with protobuf
- [x] **RTA-080**: Configurable batch sizes and intervals

## Quality Assurance Requirements

### ✅ Testing Strategy
- [x] **RTA-081**: Unit tests for MongoDB agent functionality
- [x] **RTA-082**: Integration tests with live MongoDB instances
- [x] **RTA-083**: Slow query capture validation
- [x] **RTA-084**: Error handling and recovery testing
- [x] **RTA-085**: Agent lifecycle testing

### ✅ Error Handling
- [x] **RTA-086**: Graceful degradation on MongoDB connection issues
- [x] **RTA-087**: Panic recovery in critical code paths
- [x] **RTA-088**: Comprehensive logging for debugging
- [x] **RTA-089**: User-friendly error messages in UI
- [x] **RTA-090**: Automatic retry mechanisms

### ✅ Monitoring & Observability
- [x] **RTA-091**: Prometheus metrics for query collection
- [x] **RTA-092**: Performance monitoring for collection duration
- [x] **RTA-093**: Agent status reporting and health checks
- [x] **RTA-094**: Debug logging for troubleshooting
- [x] **RTA-095**: Raw operation data preservation for analysis

## Development Infrastructure Requirements

### ✅ Build & Deployment
- [x] **RTA-096**: Docker Compose setup for development environment
- [x] **RTA-097**: Makefile targets for code generation and testing
- [x] **RTA-098**: Protobuf code generation automation
- [x] **RTA-099**: Development environment documentation
- [x] **RTA-100**: Nginx configuration for production deployment

### ✅ Documentation
- [x] **RTA-101**: API documentation via protobuf definitions
- [x] **RTA-102**: Code comments and documentation
- [x] **RTA-103**: Development setup instructions
- [x] **RTA-104**: Configuration reference
- [x] **RTA-105**: Troubleshooting guides

## Database Schema Requirements

### ✅ Schema Changes
- [x] **RTA-106**: Add `real_time_analytics_options` JSONB field to `agents` table
- [x] **RTA-107**: Database migration for schema updates
- [x] **RTA-108**: Backward compatibility maintenance
- [x] **RTA-109**: Proper indexing for performance

## Future Extensibility Requirements

### ✅ Multi-Database Support Preparation
- [x] **RTA-110**: Generic protobuf message structure
- [x] **RTA-111**: Database-specific field containers
- [x] **RTA-112**: Extensible agent architecture
- [x] **RTA-113**: Plugin-style database support framework

### ✅ Feature Enhancement Readiness
- [x] **RTA-114**: Modular component architecture
- [x] **RTA-115**: Configurable collection strategies
- [x] **RTA-116**: Extensible UI component system
- [x] **RTA-117**: API versioning support

## Known Limitations & Technical Debt

### Areas for Future Improvement
- [ ] **RTA-118**: Support for additional database types (MySQL, PostgreSQL)
- [ ] **RTA-119**: Advanced query analysis and recommendations
- [ ] **RTA-120**: Query performance trend analysis
- [ ] **RTA-121**: Historical query replay capabilities
- [ ] **RTA-122**: Advanced filtering and search features
- [ ] **RTA-123**: Export functionality for query data
- [ ] **RTA-124**: Integration with alerting systems
- [ ] **RTA-125**: Query plan analysis and visualization
- [ ] **RTA-126**: Multi-instance aggregation views
- [ ] **RTA-127**: Real-time query killing capabilities

## Completion Status

**Overall Progress: 95% Complete** ✅

### Fully Implemented Areas:
- Core RTA functionality for MongoDB
- Complete UI dashboard
- Agent management and lifecycle
- Security and access control
- CLI integration
- Development infrastructure
- Basic testing suite

### Remaining Work:
- Extended database support
- Advanced analytics features
- Performance optimization refinements
- Additional test coverage
- Production hardening

---

*Last Updated: January 2025*
*Feature Status: Production Ready for MongoDB*
