# Operations: writing an operation kind

This guide is for a feature epic that adds a managed-cluster operation (install, import,
sync, release, uninstall, user management, backup). The lifecycle framework in
`managed/services/om` runs the operation for you: a feature does not write driver code,
a state machine or SQL. It registers one **operation kind** - a declaration of knobs the
driver reads - and supplies a step catalogue and a check set. This document is the
contract that declaration must satisfy.

The framework ships no feature kind. The first real kind is install readiness (E0-S1);
this guide and a registered test kind (see [Worked example](#worked-example)) are what
later kinds follow.

## The model in one paragraph

A handler inserts one `checking` row and returns; it never runs checks or dispatches
work. On the HA leader, a single fenced driver (`Runner`, `operation_runner.go`) claims
non-terminal rows each tick and advances them one phase: it runs the kind's **check set**,
routes the result (block, await confirmation, or run), takes the kind's **lock set**
through the start gate, calls the kind's `onEnterRunning` hook inside the enter-running
transaction, then dispatches the kind's **steps** one at a time under a version-and-epoch
fence and records each result. Every mutation is a `version = version + 1 WHERE version =
$seen` CAS, so a second writer that saw the same version is fenced. All state is in
PostgreSQL; a failover resumes from the row.

## Register a kind

A kind is an `OperationKind` (`operation_kind.go`) added to the `OperationRegistry` the
`Service` holds (`WithOperationRegistry`). The driver looks a row's `kind` string up in
that registry; an unregistered kind is refused (`FailedPrecondition`), never run.

```go
reg.Register(om.OperationKind{
    Kind:                    "install",              // stored in om_operations.kind; negotiated with Extensions
    ConfirmsFor:             confirmsForTrigger,     // nil == never confirms
    CancellableWhileRunning: false,                  // true lets Cancel ask a running op to compensate
    Resumable:               true,                   // a failed op can Resume (same kind_version)
    NonInterruptibleSteps:   map[string]bool{...},   // steps a cancel must defer past
    LockSet:                 installLocks,            // func(op) []LockResource - see Locks
    Steps:                   installSteps,            // func(op) []StepPlan - the catalogue
    Deadline:                30 * time.Minute,        // bounds queued and running waits
    Attention:               installAttention,        // which attention reasons it is exempt from
    AuthMethods:             []string{...},           // the RPCs that start/confirm it (both-map, below)
    OnEnterRunning:          installOnEnterRunning,   // in-transaction entry hook
    OnSucceeded:             installOnSucceeded,      // in-transaction success hook
    OnTerminal:              installOnTerminal,       // in-transaction non-success-terminal hook
})
```

The `kind` string is the lowercase value stored in `om_operations.kind` and negotiated
with Extensions: `install`, `import`, `sync`, `release`, `uninstall`, `settings_change`.
It is **not** the API method name that starts it (`TriggerHostBootstrap`, `StartImport`,
`StartSync`, ...); a ticket that says "`StartSync`" means the API method for kind `sync`.

## The driver and its seams

The driver (`Runner`) depends on three injected interfaces (`deps.go`), so a feature
plugs its real implementations in without touching the loop; tests use fakes:

| Seam | Supplies | Filled by |
|------|----------|-----------|
| `CheckProvider.CheckInputs(ctx, op, kind)` | the kind's `CheckSet`, the current facts and their ages | E0-T4 (scans / health projection) |
| `StepExecutorProvider.Executor(op, kind)` | the `StepExecutor` that runs steps on the host | E0-T3 (run contract) |
| `ScanRequester.RequestScan(opID, profile, key)` | a scan request for a stale or over-age report | E0-T4 |

A feature kind never calls these directly. It declares the facts (in its `CheckSet`) and
the steps (in `Steps`); the provider the running server wires in produces the values.

## Check sets and stale reports

A `CheckSet` (`operation_checkset.go`) is the kind's declared `Facts` and blocking
`Rules`. Each `CheckFact` names a fact, its `MaxAge`, and whether it is `Secret`. A
`BlockingRule` reads a `CheckContext` (the cluster id and the non-secret fact values) and
returns a `RuleResult` that either passes or names what it blocks (`Blocked`, `Reason`,
`Path`).

- `report_digest` is computed by pmm-managed over exactly the declared **non-secret** fact
  values (`computeReportDigest`). A secret-typed fact that appears in the facts map is
  **rejected**, so it can never enter the digest, the report, or a log.
- The driver reads facts from the fact source **before** the lock transaction, computes the
  digest, and inside the start-gate transaction recomputes and compares. A digest mismatch
  or an over-age fact issues a scan request through `ScanRequester` and returns the
  operation to `checking` (`CheckRescanRequested`); E0-T4 fulfils the request, a fake
  records it in tests. This ticket writes no scan row.

The first real check set, install readiness, is E0-S1.

## The start gate

Every edge into `running` (from `checking`, `queued`, `awaiting_confirmation`) goes
through the one `startGate` (`operation_startgate.go`), in one transaction, in this order:

1. **Opening CAS** - `SET version = version + 1 WHERE status = <from> AND version = $seen`,
   and for a confirm **also** `AND report_digest = $seen AND params_revision = $seen`. This
   single bump claims the attempt; it does not yet set the final status.
2. **Acquire the lock set** via `om_acquire_locks` (the gate is the only caller).
3. **Attention gate** - a local read of active `om_attention` under the held locks (below).
4. **Recheck** under the locks - re-read base state and re-run the blocking rules against
   the pre-transaction facts snapshot; a confirm binds the sealed `params` revision and does
   **not** re-read live settings.
5. Set `running` and commit.

Two failures are distinct, and a feature author must understand both:

- **Opening CAS matches zero rows** (`StartRejected`) - the caller's reviewed view is stale
  or a second writer advanced the row. **Nothing changes**; the revision stays valid, so a
  fresh confirm against the current view still works.
- **Stale input under the locks** (`StartStaleInput`) - the CAS passed but the re-read shows
  the world moved. The same transaction **releases the locks it took**, returns the operation
  to `checking`, and **invalidates the revision** (bumps `params_revision`) so a replay of
  the old confirm cannot apply; it does not roll back.

A lock conflict (`om_acquire_locks` raises) aborts the transaction and a follow-up sets
`queued` (`StartQueued`); the attention gate blocking returns the op to `checking`
(`StartBlockedAttention`). `ResumeOperation` is the exception to the gate: same
`kind_version`, it keeps its existing locks, re-acquires nothing, and takes no new plan
snapshot.

## Locks

Acquire **only** through `om_acquire_locks`, and only in the start-gate transaction - never
a bare `SELECT ... FOR UPDATE` (it races the `om_locks_check` trigger). A kind's `LockSet`
returns `[]LockResource{{Kind, ID, Mode}}` with `Mode` `OmLockExclusive` or `OmLockShared`.
Mutating kinds take **exclusive** locks; **which** resources is kind-specific:

| Kind | Locks |
|------|-------|
| `install` | the **nodes** (no cluster or members exist yet; members are written at success) |
| `import` | preallocated **member** rows (the sealed plan carries the cluster and member UUIDs) |
| `sync` / `release` / `uninstall` | the **cluster + its existing members** |
| a member Sync *adds* | the **cluster (exclusive) + the target node** - no member lock (the row does not exist yet; it is inserted under the cluster lock) |

**Release matrix:** the generic lifecycle releases locks on `succeeded`, `cancelled`,
`rolled_back`, `abandoned`, and on `failed` **when the kind does not resume**. A resumable
`failed` **keeps** its locks; `ResumeOperation` does not re-acquire.

For a writer that is not an operation (a PMM backup job), use the **external hold API**
(`AcquireExternalHold` / `HeartbeatExternalHold` / `ReleaseExternalHold`,
`operation_external.go`): it inserts a row directly as `external` and takes the locks
idempotently (keyed by the job id). The driver never claims an `external` row.

## Confirmation

When a kind confirms (`ConfirmsFor` returns true for the trigger), the driver seals an
immutable plan revision in `params` when the report is stored - including the declared
settings snapshot (a secret setting contributes a reference only, never a decrypted value).
`ConfirmOperation` carries the reviewed `version`, `report_digest` **and**
`params_revision`; the opening CAS checks all three under `status = 'awaiting_confirmation'`.
Sealing the plan at report time is what stops scope widening between review and run: the
confirm binds the reviewed plan, not whatever settings are live at confirm.

## Hooks

Three in-transaction hooks (`OperationHook`, `operation_kind.go`) let a kind act without
patching the driver. Each **must be idempotent** (recovery re-runs transactions):

- **`OnEnterRunning`** runs inside the enter-running (start-gate) transaction, after the
  locks are taken, before any step. A create-kind writes its cluster/replica-set/member
  rows, the initial `pending` secrets and `planned` owned items here. The hook does **not**
  acquire locks (the gate already did).
- **`OnSucceeded`** runs in the transaction that moves the operation to `succeeded`.
- **`OnTerminal`** runs on every non-success terminal - `cancelled` (including the
  administrator-secret expiry cancel), `rolled_back`, `abandoned`, and a **non-resumable**
  `failed` (a resumable `failed` keeps everything for `Resume`) - so a kind can compensate:
  delete its own secrets and **soft-delete** the rows it created (stamp `removed_at` on the
  cluster **and** its replica sets and members, set `status = removed`; never `DELETE`).

### Contracts the schema does not enforce

A feature author must uphold these; no constraint will catch a violation:

- The plan the driver acts on is the immutable `params` revision bound to the operation,
  **never** `report`.
- Confirmation re-reads base state and the named rows by their live unique key under the
  lock; an owned item a create expects to be **absent** is one of those keys.
- Drafts are never live rows.
- A password is write-ahead to a `pending` secret (encrypted via E0-T1's helper), never
  inlined into `params`, `report`, a step `result`, or a log.

## Steps and result mapping

`Steps` returns the catalogue as `[]StepPlan{{Phase, Step, TargetKind, TargetID}}`. The
driver dispatches the next not-yet-succeeded step under the fence, runs it through the
`StepExecutor`, and maps the typed result (`operation_step.go`) onto the step status:

| `StepResultKind` | step status | meaning |
|------------------|-------------|---------|
| `StepResultOk` | `succeeded` | effect done |
| `StepResultFailed` | `failed` | effect may have occurred; the proof probe decides retryability |
| `StepResultBlocked` | `failed` (false postcondition) | a host-side precondition was not met; operation becomes resumable `failed`, records `operation_failed` |
| `StepResultTimedOut` / `StepResultUnknown` | `unknown` | inconclusive; re-probed, resolved to `succeeded` or `failed` |

There is no `blocked` step status. A new `attempt + 1` is opened **only** after a committed
`failed` whose proof shows no effect. Recovery of a committed-`dispatching` step **re-sends
the same idempotency key** and reuses that attempt's pending secret; it never opens
`attempt + 1`.

## Concurrency and fencing (what you rely on)

- **CAS on every mutation.** Zero rows updated means stop (invariant C6).
- **Leader epoch.** The dispatch transaction binds both `om_operations.version` and the live
  `om_leader_lease` epoch (`EXISTS (SELECT 1 FROM om_leader_lease WHERE epoch = $cached)`) and
  holds the lease row `FOR SHARE`. A deposed leader is fenced. The guarantee is
  **at-least-once with one logical effect** (reconciled by the idempotency key, the proof
  probe, and Extensions' lower-epoch refusal), not a single asserted invocation.

Leadership itself is the existing HA leader check (the `Runner`'s `WithLeader` gate); the
lease row is only the fencing epoch the winner advances.

## Auth: default-deny across both maps

Operations methods are fail-closed. Among them only `ListOperations` and `GetOperation` are
viewer-readable; `Confirm`, `Cancel`, `Resume`, `Abandon`, and **every operation
trigger/create** require admin. Register a kind's start/confirm methods in **both** auth
maps in `managed/services/grafana/auth_server.go`:

- the gRPC `rules` map (`/om.v1.OmService/<Method>: admin`), and
- the HTTP `methodRules` map (`POST /v1/om/<route>: admin`).

A method present in one map but not the other, or newly added without an entry, must be
admin-only. A test enumerates every Operations RPC/route and **fails if any is
unclassified**. (The settings methods and the global "unclassified defaults to admin" test
are E0-T1's.)

## No secret in observable rows or logs

The secret value and any plain (unkeyed) hash of it appear in none of `params`, `report`,
`report_digest`, a step `result`, `error_code` / `error_fields`, `om_attention` detail or
evidence, or any history API response. Convert a host/Extensions error to a stable code plus
allowlisted structured fields at ingress, and apply the same at every sink, in logs and in
telemetry. `GetOperation`'s report is **redacted for viewers** (admin sees detail).

## Step guards (measured, from E0-SP1)

A step catalogue declares each step's privilege, guards and proof of effect. The guards
below are the ones live runs surfaced; a stop/start kind must carry them:

- **Stop guard.** Block a member stop unless the unit's `TimeoutStopSec` exceeds mongod's
  clean-shutdown quiesce plus a margin (~15 s on 8.0 for a data-bearing member, ~0.9 s for an
  arbiter); otherwise the stop ends in SIGKILL. Find the unit from mongod's PID.
- **Unclean stop is `Failed` with postcondition "member down" true** - recovery routes it to
  **attention, not a retry** (retry is only `failed` + a false postcondition).
- **A `shutdown` refused because no secondary has caught up is `Blocked`, never retried with
  `force`** (forcing can roll back writes the old primary alone held).
- **A stop while a running PBM op holds the exclusive lock is `Blocked`.** Continuous PITR
  capture does **not** block a generic stop (PBM tolerates a member leaving during capture);
  only `UninstallCluster` handles PITR, in its own check (E0-S6). Re-read the PBM-state fact
  host-side immediately before the stop; the stored fact is a pre-transaction snapshot.
- **Destroy-intent exception (uninstall).** A step tearing a member down declares its
  postcondition as merely "mongod is down" and is exempt from the stop guard: a SIGKILL is
  acceptable, an unclean stop satisfies the postcondition, and a `supervised` member is
  handled by disabling its unit first (or removing the binary after the stop).
- **Arbiters:** any local user can stop an arbiter with an unauthenticated `shutdown`, so an
  arbiter found down is **not** evidence of a PMM step; start one only through its unit.

### Privilege-capability mapping

Map a step's privilege to the measured capability facts (E0-T3/E0-T4): member stop/start
needs `host.unit_control`, install needs `host.package_install`, adding a member needs
`member.keyfile_readable`. A failed mapping disables the capability and names the blocker
(`NoNewPrivileges` first).

## Worked example

The registered **test kind** in the om package's tests is a minimal, complete kind: a
cluster-exclusive `LockSet`, a one-step `Steps` catalogue, no confirmation. It is what the
end-to-end and two-driver tests drive through stale confirm, lock conflict, in-flight
re-send, resume-keeps-locks, abandon-releases-locks, and no-secret-in-history. Copy its
shape, then replace the lock set, steps, check set and hooks with your feature's.

## Each story's duty

Each feature story that adds a kind:

1. registers the kind and its auth-map entries (both maps),
2. ships its check set (declared facts and ages), step catalogue (privilege, guards, proof),
   lock set, hooks, settings sections, secret purposes, owned-item kinds and accepted-state
   paths, and
3. adds its **how-to-test** to the API reference.
