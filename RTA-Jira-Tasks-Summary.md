# Real-Time Analytics Jira Tasks Summary

## Epics Created

### Epic 1: Real-Time Analytics MVP (MongoDB Support)
**Jira ID:** PMM-14550  
**Link:** https://perconadev.atlassian.net/browse/PMM-14550  
**Status:** Open  
**Priority:** Medium

### Epic 2: Real-Time Analytics - Enhanced Features & Security
**Jira ID:** PMM-14601  
**Link:** https://perconadev.atlassian.net/browse/PMM-14601  
**Status:** Open  
**Priority:** Medium

---

## MUST HAVE Tasks - MVP (Epic: PMM-14550)

### Backend Tasks

#### Task 1: [BE] Define Protobuf Schema for Real-Time Query Data
**Jira ID:** PMM-14552  
**Link:** https://perconadev.atlassian.net/browse/PMM-14552  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Added: `query_text` field must be optional to support privacy mode
- Added: Message includes query json as is (raw query representation)
- Updated implementation to make query_text optional in protobuf

#### Task 2: [BE] Create Dedicated gRPC Endpoint for Real-Time Data
**Jira ID:** PMM-14553  
**Link:** https://perconadev.atlassian.net/browse/PMM-14553  
**Status:** Created  
**Epic:** PMM-14550

#### Task 3: [BE] Implement In-Memory Data Store on Server
**Jira ID:** PMM-14554  
**Link:** https://perconadev.atlassian.net/browse/PMM-14554  
**Status:** Created  
**Epic:** PMM-14550

#### Task 4: [BE] Create Short Polling HTTP API Endpoint via gRPC Gateway
**Jira ID:** PMM-14555  
**Link:** https://perconadev.atlassian.net/browse/PMM-14555  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Removed pagination from acceptance criteria
- Clarified service and cluster are optional with AND combination
- Filtering logic moved to separate task (Task 8)

#### Task 5: [BE] Implement MongoDB Data Collection Agent
**Jira ID:** PMM-14556  
**Link:** https://perconadev.atlassian.net/browse/PMM-14556  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Agent path: `agent/agents/mongodb/realtime/`
- Polling interval provided by pmm-server to pmm-agent (not user-configurable)
- Implements panic recovery to prevent pmm-agent crashes

#### Task 6: [BE] Implement Agent-to-Server Data Transmission
**Jira ID:** PMM-14557  
**Link:** https://perconadev.atlassian.net/browse/PMM-14557  
**Status:** Created  
**Epic:** PMM-14550  
**Note:** Reuses existing connection between pmm-agent and pmm-managed

#### Task 7: [BE] Agent Configuration and Lifecycle Management
**Jira ID:** PMM-14561  
**Link:** https://perconadev.atlassian.net/browse/PMM-14561  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Polling interval is provided by pmm-server (not user-configurable)

#### Task 8: [BE] Implement Basic Server-Side Filtering
**Jira ID:** PMM-14562  
**Link:** https://perconadev.atlassian.net/browse/PMM-14562  
**Status:** Created  
**Epic:** PMM-14550

#### Task 16: [BE] Create Database Schema for Feature Configuration
**Jira ID:** PMM-14567  
**Link:** https://perconadev.atlassian.net/browse/PMM-14567  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Three options provided: Dedicated Table, Add Column to Services Table, Create Record in Agents Table
- Developer decides which approach to use
- Note added: for Options 1 and 3, credentials must be retrieved from existing agents

#### Task 17: [BE] Implement Real-Time Analytics Configuration API
**Jira ID:** PMM-14574  
**Link:** https://perconadev.atlassian.net/browse/PMM-14574  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Dedicated `/v1/realtime/change` endpoint using gRPC Gateway
- Supports both `service_id` and `cluster` in request (using protobuf `oneof`)
- Cluster option enables RTA for all services in that cluster
- Empty response for now

#### Task 18: [BE] Implement pmm-managed Configuration Propagation
**Jira ID:** PMM-14575  
**Link:** https://perconadev.atlassian.net/browse/PMM-14575  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Reuses existing agent configuration handler in `supervisor.go`
- Uses existing `SetState` mechanism

#### Task: [BE] Implement Running RTA Agents List API
**Jira ID:** PMM-14616  
**Link:** https://perconadev.atlassian.net/browse/PMM-14616  
**Status:** Created  
**Epic:** PMM-14550  
**Key Details:**
- REST endpoint `/v1/realtime/agents` via gRPC Gateway
- Returns list of running RTA agents with: agent_id, service_id, service_name, cluster, started_at, status
- Supports filtering by cluster (optional)
- Data fetched from DB and in-memory registry
- Accessible by viewer role (read-only)
- LBAC out of scope for now

### Frontend Tasks

#### Task FE-1: [FE] Add RTA Tab and Independent Page
**Jira ID:** PMM-14587  
**Link:** https://perconadev.atlassian.net/browse/PMM-14587  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- RTA route: `/rta` (not `/qan/rta`)
- RTA is independent component at same level as QAN
- Two tabs: "QAN" and "Real-Time Analytics"

#### Task FE-2: [FE] Implement Cluster/Service Selection Page
**Jira ID:** PMM-14588  
**Link:** https://perconadev.atlassian.net/browse/PMM-14588  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Admin/editor users see all clusters and services
- Viewers only see clusters/services with active RTA
- Yellow banner and modal visible only for admin/editor users
- Cluster/Service dropdown is searchable

#### Task FE-3: [FE] Implement Running Agents Modal
**Jira ID:** PMM-14589  
**Link:** https://perconadev.atlassian.net/browse/PMM-14589  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Removed "Add another" buttons
- Can select cluster and service within cluster

#### Task FE-4: [FE] Implement RTA Table Page with Live Updates
**Jira ID:** PMM-14590  
**Link:** https://perconadev.atlassian.net/browse/PMM-14590  
**Status:** Created  
**Epic:** PMM-14550

#### Task FE-5: [FE] Implement Query Details Panel with Tabs
**Jira ID:** PMM-14591  
**Link:** https://perconadev.atlassian.net/browse/PMM-14591  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Removed Explain Plan tab for now

#### Task FE-6: [FE] Implement Pause/Resume Functionality
**Jira ID:** PMM-14592  
**Link:** https://perconadev.atlassian.net/browse/PMM-14592  
**Status:** Created  
**Epic:** PMM-14550

#### Task FE-7: [FE] Implement Cluster/Service/Node Switching
**Jira ID:** PMM-14593  
**Link:** https://perconadev.atlassian.net/browse/PMM-14593  
**Status:** Created  
**Epic:** PMM-14550

### Documentation Tasks

#### Task 21: [DOC] Create API Documentation
**Jira ID:** PMM-14581  
**Link:** https://perconadev.atlassian.net/browse/PMM-14581  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Includes documentation for API to readme.io
- Covers only endpoints 2 and 3

#### Task 22: [DOC] Write User Documentation
**Jira ID:** PMM-14586  
**Link:** https://perconadev.atlassian.net/browse/PMM-14586  
**Status:** Created  
**Epic:** PMM-14550

### QA Tasks

#### Task 20: [QA] Create End-to-End Integration Tests
**Jira ID:** PMM-14602  
**Link:** https://perconadev.atlassian.net/browse/PMM-14602  
**Status:** Created  
**Epic:** PMM-14550  
**Key Changes:**
- Simplified to let QA experts decide implementation details

---

## ENHANCED FEATURES Tasks (Epic: PMM-14601)

### Task 23: [BE] Implement Label-Based Access Control (LBAC)
**Jira ID:** PMM-14607  
**Link:** https://perconadev.atlassian.net/browse/PMM-14607  
**Status:** Created  
**Epic:** PMM-14601  
**Key Details:**
- Real-time data endpoint added to list of endpoints with filter headers
- Filters extracted from request metadata (labels in PromQL format)
- Performant in-memory filtering algorithm with hash maps
- PromQL-based filter support
- Permission caching with TTL

### Task 27: [BE] Implement Privacy Control Configuration
**Jira ID:** PMM-14608  
**Link:** https://perconadev.atlassian.net/browse/PMM-14608  
**Status:** Created  
**Epic:** PMM-14601  
**Key Details:**
- RTA respects existing `disable_query_text` from QAN agent
- PMM Managed retrieves setting from QAN and provides to RTA agent
- When privacy mode enabled: query fingerprint without sensitive data sent instead of query_text
- Cleanup query text related fields in raw JSON when privacy mode enabled
- Protobuf `query_text` field is optional (updated in Task 1)

---

## CANCELLED Tasks

### Old UI Tasks (Replaced with new UI flow)
- **Task 9:** Create Real-Time Analytics Page/View (PMM-14563) - CANCELLED
- **Task 10:** Implement Query List Display Component (PMM-14564) - CANCELLED
- **Task 11:** Implement Sorting Functionality (PMM-14565) - CANCELLED
- **Task 12:** Implement Filtering by Cluster/Service - CANCELLED
- **Task 13:** Implement Pause/Resume Functionality - CANCELLED
- **Task 14:** Implement Query Details Modal/View - CANCELLED
- **Task 15:** Implement Short Polling Client Logic - CANCELLED
- **Task 19:** Create UI Toggle Control - CANCELLED

### Bulk Operations (Skipped)
- **Task 24:** Add Real-Time Metric Graphs - CANCELLED
- **Task 25:** Implement Bulk Selection in Inventory Page - CANCELLED
- **Task 26:** Implement Bulk Enable/Disable Operations API - CANCELLED

### Security Tasks (Already covered in LBAC or Must Have)
- **Task 28:** Implement Authentication for Real-Time API - CANCELLED (covered by existing auth)
- **Task 29:** Add Authorization Checks - CANCELLED (covered by LBAC Task 23)

---

## PENDING Tasks (Not Yet Created)

### Enhanced Features Epic (PMM-14601)
- **Task 30:** [QA] Create Performance Test Suite
- **Task 31:** [DOC] Write Developer Documentation

### Could Have Features (No Epic Yet)
- **Task 32:** Enhance Server-Side Filtering
- **Task 33:** Add Advanced Filtering by Labels
- **Task 34:** Implement Advanced Visualizations
- **Task 35:** Add Short-Term History View
- **Task 36:** Display Status on Inventory Page
- **Task 37:** Add Advanced Configuration Options
- **Task 38:** Implement CLI Control via pmm-admin
- **Task 39:** Implement Automatic Start/Stop Logic (SKIPPED FOR NOW)
- **Task 40:** Implement Server-Side Monitoring
- **Task 41:** Implement Agent-Side Monitoring
- **Task 42:** Add Alerting Integration
- **Task 43:** Create Video Tutorials

### Future Enhancements (No Epic Yet)
- **Task 44:** Implement WebSocket Communication
- **Task 45:** Add MySQL Support
- **Task 46:** Add PostgreSQL Support
- **Task 47:** Implement Adaptive Polling
- **Task 48:** Include Finished Queries

---

## Summary Statistics

### Created Tasks: 25
- **MVP Epic (PMM-14550):** 22 tasks
  - Backend: 12 tasks
  - Frontend: 7 tasks
  - Documentation: 2 tasks
  - QA: 1 task
- **Enhanced Features Epic (PMM-14601):** 3 tasks
  - Backend: 2 tasks (LBAC, Privacy Control)
  - QA: 0 tasks (pending)
  - Documentation: 0 tasks (pending)

### Cancelled Tasks: 11
- Old UI tasks: 8
- Bulk operations: 3
- Security (covered elsewhere): 2

### Pending/Not Created: 19 tasks
- Enhanced Features: 2
- Could Have: 12
- Future: 5

---

## Key Implementation Notes

1. **Privacy Control:** Query text is optional in protobuf, supports both full text and fingerprint modes
2. **Authentication/Authorization:** Handled by existing PMM infrastructure and LBAC (Task 23)
3. **Agent Path:** `agent/agents/mongodb/realtime/`
4. **API Endpoints:**
   - `/v1/realtime/query-data` - Get real-time query data (short polling)
   - `/v1/realtime/change` - Enable/disable RTA for service or cluster
   - `/v1/realtime/agents` - Get list of running RTA agents (viewer accessible)
5. **UI Route:** `/rta` (independent from QAN)
6. **Configuration:** Stored in database (three options provided to developer)
7. **Agent Communication:** Reuses existing pmm-agent to pmm-managed connection
8. **Polling Interval:** Provided by pmm-server, not user-configurable (for now)

---

## Next Steps

1. Decision needed: Create "Could Have" epic or add remaining tasks to existing epics?
2. Create Task 30 (Performance Test Suite) and Task 31 (Developer Documentation)?
3. Decide which "Could Have" features to create as Jira tasks
4. Decide which "Future" features to create as Jira tasks

---

*Last Updated: 2025-12-04*

