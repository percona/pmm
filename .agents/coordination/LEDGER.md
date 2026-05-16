# Agent Coordination Ledger — pmm

> Protocol: `.agents/skills/agent-coordination/SKILL.md`
> Timestamps: UTC ISO-8601 (`date -u +%Y-%m-%dT%H:%M:%SZ`)
> STALE threshold: `last_heartbeat` > 30 min → reclaimable
> This file is committed to Git; `git pull` synchronizes it across all machines and agents.

## Agents

| agent_id | host | status | last_heartbeat | notes |
|---|---|---|---|---|
| tungs-cbbf524f | tungs | ACTIVE | 2026-05-16T16:59:02Z | PMM realignment to v3.7.1 + coordination protocol adoption |

## Area Locks

| area | agent_id | acquired_at | lease_expires_at | purpose |
|---|---|---|---|---|
| Makefile.datacosmos, .agents/, build pipeline, agent/agents/clickhouse/ | tungs-cbbf524f | 2026-05-16T16:16:22Z | 2026-05-16T17:30:00Z | PMM realignment — release after datacosmos/build push |

## Tasks

| task_id | title | owner | status | claimed_at | due | last_update | notes |
|---|---|---|---|---|---|---|---|
| PMM-ALIGN-001 | Realign PMM fork to upstream v3.7.1; create fork; isolate ClickHouse draft; fix build pollution | tungs-cbbf524f | IN_PROGRESS | 2026-05-16T16:16:22Z | 2026-05-16T18:16:22Z | 2026-05-16T16:59:02Z | fork datacosmos-br/pmm created; feat/clickhouse-collector built+pushed; datacosmos/build being committed. Root-cause fixes: removed in-repo build-root/ (7.7GB) + stale untracked vendor/ + stray bin/*.go that broke `go build ./...`. |
| AGENTS-COORD-001 | Adopt & standardize multi-agent coordination protocol | tungs-cbbf524f | IN_PROGRESS | 2026-05-16T16:16:22Z | 2026-05-16T18:16:22Z | 2026-05-16T16:59:02Z | Ledger + skill + AGENTS.md §W-9 on datacosmos/build. PENDING (human): centralized `~/.agents` R5/R11 edits + `git init` — agent blocked by self-modification guard. |

## Log

| timestamp | agent_id | scope | message |
|---|---|---|---|
| 2026-05-16T16:16:22Z | bootstrap | init | Ledger initialized. Protocol adopted per agent-coordination SKILL.md §9. |
| 2026-05-16T16:16:22Z | tungs-cbbf524f | heartbeat | Registered ACTIVE. No prior ledger existed. |
| 2026-05-16T16:16:22Z | tungs-cbbf524f | task claim | task=PMM-ALIGN-001 + AGENTS-COORD-001 |
| 2026-05-16T16:45:00Z | tungs-cbbf524f | progress | task=PMM-ALIGN-001 root-cause: `go build ./...` cascade was an in-repo build-root/ (7.7GB module cache), a stale untracked vendor/ shadowing v3.7.1 module mode, and stray bin/*.go. All moved to ../pmm-cleanup-bak/. |
| 2026-05-16T16:55:00Z | tungs-cbbf524f | progress | task=PMM-ALIGN-001 done=fork created; feat/clickhouse-collector off v3.7.1 builds clean (vet+gofmt ok) and pushed. clickhouse-go/v2 already an upstream dep — no go.mod change. |
| 2026-05-16T16:59:02Z | tungs-cbbf524f | heartbeat | still active; tasks=PMM-ALIGN-001,AGENTS-COORD-001; committing datacosmos/build next. |
