# Investigations data model and API contract

This document defines the investigations data model and REST API contract for the PMM AI Investigations feature.

## Data model

### investigations

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| title | text | Investigation title |
| status | text | open, investigating, resolved, archived |
| severity | text | Optional severity |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| created_by | text | User id or empty |
| time_from | timestamptz | Incident time window start |
| time_to | timestamptz | Incident time window end |
| summary | text | Short "what happened and why" (3–4 lines), shown at top |
| summary_detailed | text | Optional; longer narrative shown later |
| root_cause_summary | text | Optional |
| resolution_summary | text | Optional |
| source_type | text | manual, alert, scheduled, ai |
| source_ref | text | e.g. alert fingerprint |
| tags | JSONB/array | Optional |
| config | JSONB | Optional service/cluster refs |

### investigation_blocks

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| investigation_id | FK | References investigations |
| type | text | summary, timeline, single_panel, panel_group, logs_view, query_result, finding, markdown, slow_query_analysis, top_queries, schema_view, comment_thread, chat_thread, attachments, remediation_steps, … |
| title | text | Optional block title |
| position | integer | Order (gaps allowed) |
| config_json | JSONB | e.g. dashboard_uid, panel_id, time range for panels |
| data_json | JSONB | Block payload |
| created_at | timestamptz | |
| updated_at | timestamptz | |
| created_by | text | Optional |
| updated_by | text | Optional |

### investigation_artifacts

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| investigation_id | FK | References investigations |
| type | text | panel_snapshot, query_result, log_excerpt, report_pdf, ai_finding |
| uri_or_blob_ref | text | Reference to stored artifact |
| source | text | Optional |
| metadata_json | JSONB | Optional |
| created_at | timestamptz | |

### investigation_messages

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| investigation_id | FK | References investigations |
| role | text | user, assistant, tool |
| content | text | Message content |
| tool_name | text | Nullable; if role=tool |
| tool_result_json | text/JSONB | Nullable |
| created_at | timestamptz | |

### investigation_comments

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| investigation_id | FK | References investigations |
| block_id | FK | Nullable; comment on a block |
| anchor_json | JSONB | Nullable; selection range for "highlight and comment" |
| author | text | |
| content | text | |
| created_at | timestamptz | |
| updated_at | timestamptz | |

### investigation_timeline_events

| Column | Type | Description |
|--------|------|-------------|
| id | UUID/ULID | Primary key |
| investigation_id | FK | References investigations |
| event_time | timestamptz | |
| type | text | |
| title | text | |
| description | text | Optional |
| source | text | Optional |
| metadata_json | JSONB | Optional |

## API contract

### Investigation lifecycle

- `POST /v1/investigations` — Create. Body: title, time_from, time_to, source_type, source_ref, optional summary. Returns full investigation.
- `GET /v1/investigations` — List. Query: status, limit, offset. Returns list with id, title, status, created_at, updated_at, time_from, time_to.
- `GET /v1/investigations/:id` — Get one (full investigation + blocks + optional latest messages count).
- `PATCH /v1/investigations/:id` — Update. Body: title, status, summary, root_cause_summary, resolution_summary, etc.

### Blocks, timeline, artifacts, comments

- `GET /v1/investigations/:id/blocks` — Ordered blocks.
- `POST /v1/investigations/:id/blocks` — Add block. Body: type, title, position, config_json, data_json.
- `PATCH /v1/investigations/:id/blocks/:blockId` — Update block.
- `DELETE /v1/investigations/:id/blocks/:blockId` — Remove block.
- `GET /v1/investigations/:id/timeline`, `POST /v1/investigations/:id/timeline` — Timeline events.
- `GET /v1/investigations/:id/artifacts`, `POST /v1/investigations/:id/artifacts` — Artifacts.
- `GET /v1/investigations/:id/comments`, `POST /v1/investigations/:id/comments` — Comments. POST body: content, optional block_id, optional anchor_json.

### Chat and run

- `GET /v1/investigations/:id/messages` — List messages (pagination).
- `POST /v1/investigations/:id/chat` — Send message to orchestrator. Body: message, optional stream. Normal chat only (single-round Q&A unless run_full_investigation flag).
- `POST /v1/investigations/:id/run` — Run full investigation (multi-turn loop). Optional stream for progress.

### Export

- `POST /v1/investigations/:id/export/pdf` — Generate PDF of full report. Returns PDF bytes.

### Visibility

Investigations are org-scoped: all users in the same Grafana org see the same list and can open any investigation.
