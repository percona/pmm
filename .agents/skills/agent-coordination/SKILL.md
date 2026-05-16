---
name: agent-coordination
license: MIT
metadata:
  author: marlonsc
  version: '2.0.0'
description:
  'Multi-agent coordination protocol for concurrent AI agents working the same repo(s) across machines. Defines the
  committed coordination ledger (.agents/coordination/LEDGER.md): agent identity + heartbeat, area locks (token strategy)
  with leases/TTL, a task ledger with timestamps and expiry (vencimento), communication patterns, stale detection and
  takeover rules, conflict escalation, and commit discipline. Prevents collisions, undone work, and idle/stale agents.
  WHEN: starting ANY session in a repo where more than one agent may work; before editing files; before claiming or
  handing off a task; when you detect another agent diverged from your approach; before stopping/pausing work; whenever
  you commit. This protocol is MANDATORY and standardized — always applied.'
---

# Agent Coordination — concurrent agents, one repo

> **AUTHORITATIVE — MANDATORY COMPLIANCE.** Multiple AI agents (possibly on
> different machines) work these repositories concurrently. Without coordination
> they collide on files, undo each other's commits, duplicate effort, and leave
> tasks stale. This protocol is the **single, standardized mechanism** committed
> to Git — it propagates to every agent, machine, and repo clone automatically.

---

## 0. The coordination ledger — single source of truth

The live, committed file `.agents/coordination/LEDGER.md` (workspace root) is
the SSOT. Every agent reads it at session start and updates it while working.
Because it is in Git, `git pull` synchronizes it across all machines.

**Four sections:** `Agents` · `Area Locks` · `Tasks` · `Log`

All timestamps **must be UTC ISO-8601**: `date -u +%Y-%m-%dT%H:%M:%SZ`

---

## 1. Agent identity & registration

Your agent ID is stable for the session: **`<host>-<sessionShort>`**
(e.g. `tungs-8f880ad9`).

```bash
# Derive your agent ID
HOST=$(hostname -s)
SESSION_SHORT=$(echo "$CONVERSATION_ID" | cut -c1-8 2>/dev/null || head -c8 /dev/urandom | xxd -p)
AGENT_ID="${HOST}-${SESSION_SHORT}"
NOW=$(date -u +%Y-%m-%dT%H:%M:%SZ)
```

**Session start sequence (mandatory — no exceptions):**
1. `git pull` the main repo (and any relevant submodule)
2. Read `.agents/coordination/LEDGER.md`
3. Add/refresh your row in **Agents**: `status: ACTIVE`, `last_heartbeat: <now>`
4. Commit + push the ledger update: `chore(agents): register <agent_id> heartbeat`
5. Scan **Area Locks** for expired/stale locks and **Tasks** for reclaimable work

---

## 2. Heartbeat — the primary anti-stale mechanism

The heartbeat is what prevents an agent from going **STALE** and having its
work reclaimed or duplicated.

| Rule | Value |
|---|---|
| Heartbeat interval | ≤ **15 min** while ACTIVE |
| STALE threshold | `now − last_heartbeat > 30 min` |
| STALE lock lease | `lease_expires_at < now` |
| STALE task | `last_update` > 30 min ago AND status = `IN_PROGRESS` |

**When to update `last_heartbeat`:**
- Every time you write to the ledger (locks, tasks, log)
- At least every 15 min even if nothing changed (write a brief log line)
- Before every `git push` (so others see you as alive)

**Heartbeat commit format:**
```
chore(agents): heartbeat <agent_id> <now>
```

**If you cannot heartbeat (blocked/waiting):** write a log line with expected
return time:
```
| <now> | <agent_id> | heartbeat | Blocked on <reason>; will resume by <time>. Tasks <IDs> remain IN_PROGRESS. |
```

---

## 3. Area locks — the token strategy

An **Area Lock** is your exclusive write token for a path scope. No two live
agents may hold overlapping locks simultaneously.

**Area Locks table row:**
```
| area | agent_id | acquired_at | lease_expires_at | purpose |
```

### Token lifecycle

```
ACQUIRE → HOLD (≤60 min) → RENEW (if still working) → RELEASE
                        ↓ expires without renewal
                   EXPIRED (reclaimable by others)
```

**Rules:**
1. Acquire a lock before touching ANY file in the area.
2. Area scope must be **as narrow as possible** — one app dir, one appset file,
   one values group. Never lock an entire submodule unless unavoidable.
3. Lease = **60 min**. Renew by extending `lease_expires_at` + heartbeat.
4. **Expired lock** (lease passed) OR lock held by **stale agent**
   (heartbeat > 30 min) → may be taken over. Log the takeover before editing.
5. Release immediately when done with the area — do not hold locks speculatively.
6. Commit the ledger on acquire and on release (two separate commits are fine).

**Acquire pattern:**
```markdown
| apps/grafana/ | tungs-8f880ad9 | 2026-05-16T15:00:00Z | 2026-05-16T16:00:00Z | fix grafana admin ESO |
```

**Release pattern:** replace the row with `_(released)_` or delete it, commit.

---

## 4. Task ledger — timestamps & expiry (vencimento)

**Tasks table row:**
```
| task_id | title | owner | status | claimed_at | due | last_update | notes |
```

### Status machine

```
OPEN → CLAIMED → IN_PROGRESS → DONE
              ↓
           BLOCKED  (with revisit time in notes)
              ↓
           CONFLICT (owner = HUMAN; agents stop touching the area)
```

### Expiry (vencimento)

| Condition | Agent is STALE if… |
|---|---|
| Status `IN_PROGRESS` | `last_update` > **30 min** ago |
| Status `CLAIMED` | `claimed_at` > **1 h** ago, no `IN_PROGRESS` update |
| `due` field | task not `DONE` and `due < now` by > **30 min** |

A stale task may be reclaimed: update `owner`, `claimed_at`, `due`,
`last_update`, add a Log entry explaining the takeover.

### Task discipline

- Pick only `OPEN` or **STALE** tasks.
- Set a realistic `due` when claiming (default: `claimed_at + 2h`).
- Refresh `last_update` on every meaningful step (commit, progress, blocker).
- `BLOCKED` notes must say: *what blocks*, *what is needed*, *revisit time*.
- Never leave `IN_PROGRESS` without heartbeating ≤15 min.
- `DONE` must include verification evidence in `notes` (commit hash, argocd status).

---

## 5. Communication patterns — the Log

The **Log** is append-only. Every entry must be self-contained — readable by a
cold agent with no prior context.

```
| timestamp | agent_id | scope | message |
```

### Mandatory log events

| Event | Required log content |
|---|---|
| Lock **acquire** | `area=<path> purpose=<why> lease=<expires>` |
| Lock **release** | `area=<path> reason=done/timeout` |
| Lock **takeover** | `area=<path> took-from=<stale_agent> stale-since=<last_hb>` |
| Task **claim** | `task=<id> due=<when> plan=<1-line summary>` |
| Task **progress** | `task=<id> done=<what> next=<what>` |
| Task **done** | `task=<id> evidence=<commit/url/status>` |
| Task **blocked** | `task=<id> blocked-by=<reason> revisit=<time>` |
| Task **handoff** | `task=<id> done=<what> remaining=<what> verify=<how>` |
| **Conflict** open | `conflict=<id> area=<path> agent-a=<id> agent-b=<id> summary=<what diverged>` |
| Heartbeat (periodic) | `still active; tasks=<IDs>; next-hb=<time>` |

**Communication SLA (mandatory):**

- Heartbeat ≤15 min while ACTIVE.
- Task `last_update` on every meaningful step (claim/progress/blocker/done).
- Any blocker must include explicit revisit metadata (`revisit_at` timestamp or concrete revisit condition);
  any handoff must include verification commands.

### Handoff template (on pause/stop)

```
| <now> | <agent_id> | handoff | 
  DONE: <list what was committed — include commit hashes>.
  REMAINING: <list what is left with task IDs>.
  BLOCKED: <any blockers + why>.
  VERIFY: argocd app get <app> --grpc-web; kubectl get <resource>.
  NEXT AGENT: claim tasks <IDs>; area locks: none held.
```

### Blocker template

```
| <now> | <agent_id> | blocker |
  Task <id> BLOCKED: <exact error or dependency missing>.
  Unblocked by: <what needs to happen>.
  Revisit: <timestamp or condition>.
  Not progressing until unblocked — task set to BLOCKED.
```

---

## 6. Anti-stale & anti-collision rules (inviolable)

1. **Session start:** `git pull`; read ledger; register/refresh Agents row.
2. **Before editing:** acquire Area Lock; `git fetch` + rebase if behind.
3. **Never** edit a file in an area locked by a **live** (non-stale) other agent.
4. **Never** revert/overwrite another agent's commits. Divergence → CONFLICT (§7).
5. **Small batches:** one task → commit + push + validate → next task.
   Uncommitted work > 30 min = stale risk. Commit incrementally.
6. **Ledger commits** use `--rebase` before push (sections are merge-friendly).
7. **Stale reclaim**: log first (with evidence: the stale timestamp), then take over.
8. **Never claim `IN_PROGRESS` tasks** owned by a live agent — even if slow.
9. **No per-agent worktree assumption**: agents may share the same branch/worktree; coordination must
  always flow through ledger + locks + timestamps.
10. **Operational commit cadence**: ledger updates are coordination metadata and should be committed/pushed
  immediately (high frequency), without waiting for test gates when no runtime code changed.
11. **Metadata-only fast path**: edits restricted to coordination metadata (`LEDGER.md`, coordination
  sections in `AGENTS.md`/`CLAUDE.md`, `agent-coordination/SKILL.md`) must be committed/pushed immediately
  and are exempt from waiting on lint/type/test gates.

---

## 7. Conflict escalation

When agents diverge (two different fixes for the same component):

1. Create a `CONFLICT` task with both approaches described + affected files.
2. Set `owner: HUMAN` and `status: CONFLICT`.
3. Write a Log entry with both agents' approaches and why they conflict.
4. **Stop editing the conflicted area.** Do not undo the other agent's work.
5. A human decides the canonical approach; then both agents converge on that.

---

## 8. Commit discipline (propagation guarantee)

- The ledger, this skill, and the protocol anchors in `AGENTS.md`/`CLAUDE.md`
  are **committed to Git** → they propagate to every machine and repo clone.
- This skill lives in `.agents/skills/agent-coordination/SKILL.md` (workspace)
  AND `~/.agents/skills/agent-coordination/SKILL.md` (universal, same content).
- Atomic commits per task/area. `git add <specific-files>` — never `git add -A`.
- No `git push --force`, no `git reset --hard` without explicit user approval.
- After each completed task: **commit + push + validate** before starting next.
- Ledger-only commits are fine: `chore(agents): <heartbeat|lock|task|handoff>`.

---

## 9. Protocol adoption in a new repository

To adopt this protocol in any repo:

```bash
# 1. Create the directories
mkdir -p .agents/coordination .agents/skills/agent-coordination

# 2. Copy this skill
cp ~/.agents/skills/agent-coordination/SKILL.md .agents/skills/agent-coordination/SKILL.md

# 3. Create a LEDGER.md from the template below
# 4. Add §W-9 reference to AGENTS.md or CLAUDE.md
# 5. Commit everything
git add .agents/
git commit -m "chore(agents): adopt multi-agent coordination protocol"
git push
```

**LEDGER.md template:**
```markdown
# Agent Coordination Ledger

> Protocol: .agents/skills/agent-coordination/SKILL.md
> Timestamps: UTC ISO-8601 (date -u +%Y-%m-%dT%H:%M:%SZ)
> STALE threshold: last_heartbeat > 30 min → reclaimable

## Agents
| agent_id | host | status | last_heartbeat | notes |

## Area Locks
| area | agent_id | acquired_at | lease_expires_at | purpose |
| _(none)_ | — | — | — | — |

## Tasks
| task_id | title | owner | status | claimed_at | due | last_update | notes |

## Log
| timestamp | agent_id | scope | message |
| <now> | bootstrap | init | Ledger initialized. Protocol adopted. |
```

---

## Quick checklist (every session — no skipping)

```
[ ] git pull; read .agents/coordination/LEDGER.md
[ ] register/refresh Agents row (last_heartbeat = now UTC); commit + push
[ ] scan for stale agents/locks/tasks; reclaim if eligible (log first)
[ ] pick only OPEN or STALE tasks; claim with realistic due
[ ] acquire NARROW Area Lock before editing; commit ledger; lease ≤60 min
[ ] heartbeat ≤15 min: update last_heartbeat + log line; commit + push
[ ] refresh task last_update on every meaningful step
[ ] small batch: one task → commit + push + validate → next
[ ] on block: set task BLOCKED + log blocker + revisit time; heartbeat
[ ] on divergence: open CONFLICT task (owner=HUMAN), stop touching area
[ ] on stop/pause: release locks, set tasks DONE/BLOCKED, write handoff Log
[ ] last push: verify git log --oneline -3 shows all your work landed
```
