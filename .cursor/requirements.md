# Real-Time Query Analytics (RTA) Feature Requirements

## Overview
Real-Time Query Analytics (RTA) is a feature for Percona Monitoring and Management (PMM) that provides real-time visibility into currently running database queries, starting with MongoDB support.

## Core Architecture Requirements

### ✅ API & Protocol Design
- [x] Define protobuf messages for real-time analytics communication
- [x] Create gRPC service definition for agent-to-server communication
- [x] Design HTTP REST API endpoints for UI communication
- [x] Implement proper error handling and status codes
- [x] Support for both current and historical data retrieval

### ✅ Database Agent Implementation
- [x] MongoDB real-time analytics agent using `db.currentOp()` 
- [x] Collection of currently running queries with microsecond precision
- [x] Query fingerprinting using shared PMM fingerprinter
- [x] Configurable collection intervals (default: 1 second)
- [x] Query text extraction with privacy controls (disable examples)
- [x] Support for MongoDB 7.x `$currentOp` aggregation pipeline
- [x] Raw `currentOp` document preservation for debugging
- [x] Operation ID (`opid`) tracking for query deduplication
- [x] Panic recovery and error handling for robustness

### ✅ Backend Server Implementation
- [x] Real-time analytics gRPC service in pmm-managed
- [x] Data buffering with configurable retention (default: 2 minutes)
- [x] Service metadata enrichment from agent data
- [x] Label-Based Access Control (LBAC) integration
- [x] Historical vs. current data separation
- [x] Per-service data isolation and management

### ✅ Agent Management & Lifecycle
- [x] RTA agent type registration in inventory system
- [x] Agent configuration via `SetStateRequest` from pmm-managed
- [x] Automatic agent creation when adding MongoDB services
- [x] Agent status reporting and health monitoring
- [x] Connection management and error recovery
- [x] Graceful agent start/stop/restart capabilities

### ✅ Configuration Management
- [x] Store RTA configuration in JSONB field within `agent` table
- [x] Support for enable/disable per service
- [x] Configurable collection intervals
- [x] Query text privacy controls (disable examples)
- [x] Configuration validation and defaults

### ✅ Communication Infrastructure
- [x] pmm-agent to pmm-managed gRPC streaming
- [x] Efficient batch data transmission
- [x] Connection reuse and optimization
- [x] Authentication and authorization for gRPC endpoints
- [x] Nginx reverse proxy configuration for gRPC/HTTP routing

## User Interface Requirements

### ✅ Core UI Components
- [x] Real-time query analytics dashboard page
- [x] Service selection and filtering interface
- [x] Query table with sortable columns (fingerprint, database, duration, state, timestamp)
- [x] Query details dialog with raw MongoDB document display
- [x] Real-time data refresh with configurable intervals
- [x] Search and filtering capabilities
- [x] State-based query filtering (running, waiting, finished)

### ✅ Data Visualization
- [x] Query execution time display with millisecond precision
- [x] Query state indicators with color coding
- [x] Database and operation type labeling
- [x] Formatted JSON display for raw MongoDB operations
- [x] Service metadata display (node, labels)

### ✅ User Experience
- [x] Auto-refresh with 2-second intervals
- [x] Query deduplication by MongoDB `opid`
- [x] Latest query prioritization for duplicate operations
- [x] Responsive design with Material-UI components
- [x] Error handling for missing or undefined data
- [x] Loading states and empty state handling

## Security & Access Control Requirements

### ✅ Authentication & Authorization
- [x] LBAC integration following qan-api2 patterns
- [x] Service-level access control
- [x] Role-based permissions (viewer role minimum)
- [x] Secure gRPC communication with proper authentication
- [x] Query text privacy controls per service configuration

### ✅ Data Privacy
- [x] Optional query text collection (can be disabled)
- [x] Raw operation data sanitization
- [x] Service isolation and access controls
- [x] No sensitive credential exposure in logs or UI

## CLI Integration Requirements

### ✅ pmm-admin Integration
- [x] `pmm-admin add mongodb` command enables RTA by default
- [x] RTA configuration options in CLI
- [x] Agent type validation and compatibility checks
- [x] Proper error handling and user feedback

## Performance & Scalability Requirements

### ✅ Data Collection Efficiency
- [x] Minimal impact on monitored MongoDB instances
- [x] Configurable collection intervals to balance load
- [x] Efficient `$currentOp` aggregation with proper filtering
- [x] Connection pooling and reuse
- [x] Prometheus metrics for monitoring collection performance

### ✅ Data Management
- [x] Automatic data expiration (2-minute history buffer)
- [x] Memory-efficient data structures
- [x] Batch processing for efficient network usage
- [x] Separate current vs. historical data handling

### ✅ Network Optimization
- [x] gRPC streaming for real-time data
- [x] Connection reuse to minimize overhead
- [x] Efficient serialization with protobuf
- [x] Configurable batch sizes and intervals

## Quality Assurance Requirements

### ✅ Testing Strategy
- [x] Unit tests for MongoDB agent functionality
- [x] Integration tests with live MongoDB instances
- [x] Slow query capture validation
- [x] Error handling and recovery testing
- [x] Agent lifecycle testing

### ✅ Error Handling
- [x] Graceful degradation on MongoDB connection issues
- [x] Panic recovery in critical code paths
- [x] Comprehensive logging for debugging
- [x] User-friendly error messages in UI
- [x] Automatic retry mechanisms

### ✅ Monitoring & Observability
- [x] Prometheus metrics for query collection
- [x] Performance monitoring for collection duration
- [x] Agent status reporting and health checks
- [x] Debug logging for troubleshooting
- [x] Raw operation data preservation for analysis

## Development Infrastructure Requirements

### ✅ Build & Deployment
- [x] Docker Compose setup for development environment
- [x] Makefile targets for code generation and testing
- [x] Protobuf code generation automation
- [x] Development environment documentation
- [x] Nginx configuration for production deployment

### ✅ Documentation
- [x] API documentation via protobuf definitions
- [x] Code comments and documentation
- [x] Development setup instructions
- [x] Configuration reference
- [x] Troubleshooting guides

## Database Schema Requirements

### ✅ Schema Changes
- [x] Add `real_time_analytics_options` JSONB field to `agents` table
- [x] Database migration for schema updates
- [x] Backward compatibility maintenance
- [x] Proper indexing for performance

## Future Extensibility Requirements

### ✅ Multi-Database Support Preparation
- [x] Generic protobuf message structure
- [x] Database-specific field containers
- [x] Extensible agent architecture
- [x] Plugin-style database support framework

### ✅ Feature Enhancement Readiness
- [x] Modular component architecture
- [x] Configurable collection strategies
- [x] Extensible UI component system
- [x] API versioning support

## Known Limitations & Technical Debt

### Areas for Future Improvement
- [ ] Support for additional database types (MySQL, PostgreSQL)
- [ ] Advanced query analysis and recommendations
- [ ] Query performance trend analysis
- [ ] Historical query replay capabilities
- [ ] Advanced filtering and search features
- [ ] Export functionality for query data
- [ ] Integration with alerting systems
- [ ] Query plan analysis and visualization
- [ ] Multi-instance aggregation views
- [ ] Real-time query killing capabilities

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
