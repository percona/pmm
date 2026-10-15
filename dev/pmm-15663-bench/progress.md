# SDD ledger — plan: dev/pmm-15663-bench/PLAN.md

Spec: dev/pmm-15663-bench/DESIGN.md. Implementation worktree: .superpowers/wt-impl (branch PMM-15663-scoped-annotations,
from origin/PMM-15609-fleet-all-value be751c4b6). Helm: percona-helm-charts branch PMM-15663-vmauth-delete-route.
Ledger mirrored to the bench branch as dev/pmm-15663-bench/progress.md (runner moves wipe .superpowers/).

## Pre-flight scan

| Pair / task | Produces → consumes | Finding |
|---|---|---|
| T2 → T3 | `Event`, `MetricName` → Store payload | consistent |
| T2,T3 → T4 | `ObjectLabels`, `Event.Validate`, `Store.Write/Delete` → `annotationStore` | consistent (Store has both methods) |
| T4 → T5 | `annotationStore.Delete`; both edit annotation.proto, deps.go, annotation.go, grafana client | sequential, consistent |
| T4 → T6 | `AddAnnotationResponse.annotation_id` → pmm-admin output | generated client types int64 as `int64`; consistent |
| T2,T3 → T7 | `Event`, `ObjectLabels`, `Key`, `Store.Keys` → `syncStore` | consistent |
| T4 → T7 | both edit main.go | sequential |
| T4,T5 → T8 | AddAnnotation id, RemoveAnnotation → API tests | consistent |
| T1 → T9 | go/no-go on `node_name_list` | T9 waits for T1's verdict |
| T5 → T10 | delete path → vmauth route | consistent |
| T2 self | TestValidate 37+3=40 ok, 38+3=41 rejected (tags empty) | consistent |
| T3 self | 30-day window = 2592000s; base `/prometheus/` path | consistent with VMBaseURL |
| T4 self | 35 custom labels + node/service labels + 3 reserved > 40 | consistent |
| T7 self | `DatabaseDSN` returns an error; main.go must handle it | Ruling below |
| T8 self | needs live server | dev VM |
| T9 self | `test_scoped_entry_present` uses `git show origin/main` | Ruling below |
| T11 self | `make doc-build` needs make/docker | dev VM |
| T12 self | Feature Build / PRs need the user's go-ahead | stop point |
| Env | runner has no go, make, buf; DB-backed tests need PostgreSQL (forbidden on host) | Ruling below |

Ruling: All Go builds, `make gen`, unit tests, lint and docs builds run in PMM's dev container on CHAOS VM `PMM-15663-dev`
through `dev/pmm-15663-bench/devvm.sh` — the runner has no Go toolchain and must not run PostgreSQL — cost if wrong:
slower iteration only.
Ruling: Task 1 runs in parallel with Tasks 2–8 on its own VM; it only gates Task 9 — cost if wrong: none, Task 9 waits
for its verdict.
Ruling: Task 7 main.go — if `grafana.DatabaseDSN` errors, log it and do not register the sync job, as the alerting
provisioner logs and continues — cost if wrong: backfill silently off on a misconfigured HA install (logged).
Ruling: Task 9 `test_scoped_entry_present` uses a committed exclusion set (global-only dashboards and the k8s one)
instead of `git show origin/main`, because CI clones may lack `origin/main` — cost if wrong: the list needs updating
when a dashboard is added.

## Progress
Task 1: dispatched (agent a1dc53a, VM PMM-15663-bench2), runs parallel to T2-T8
