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
Task 2: dispatched (agent affd861, BASE be751c4b6)
Ruling: lint as `GOTOOLCHAIN=go1.27.1 bin/golangci-lint run ...` — the container's go1.27.2 emits export data golangci-lint v2.13.2 cannot read; go1.27.1 is go.mod's version, which CI's go_version_file selects — cost if wrong: none, CI stays the final gate.
Ruling: accept the Co-authored-by trailer the runner's global hooks put on every commit — host policy, also on earlier bench commits — cost if wrong: one trailer line per commit.
Ruling: the controller transcribed Task 2's report into task-2-report.md (the implementer declined to write .md files); impl-context now states the report file is required — cost if wrong: none.
Task 2: implementer DONE_WITH_CONCERNS, commit 9702d85c3; controller ran lint with go1.27.1: 0 issues
Task 2: minor (deferred): Co-authored-by trailer on the commit — covered by the trailer ruling above
Task 2: minor (deferred): straddle test covers only a 2-byte rune; add 3- and 4-byte cases
Task 2: minor (deferred): reserved-wins on empty value tested only for tags, not text/annotation_id
Task 2: minor (deferred): invalid-custom-labels test asserts only require.Error
Task 2: minor (deferred, final review should triage as likely must-fix): no value-length check besides text — joined `tags` or a custom label value over 4,096 bytes makes VictoriaMetrics drop the series silently; Validate could reject any value over MaxTextBytes
Task 2: complete (commits be751c4b6..9702d85c3, review clean)
Task 3: dispatched (agent ac94386, BASE 9702d85c3)
Task 3: implementer DONE_WITH_CONCERNS, commit c80f2b67a; report transcribed by controller
Ruling: Keys stays strict (errors on a missing or non-numeric annotation_id) — only pmm-managed writes this metric, a failing sync pass is logged and retried, and skipping would hide a writer bug — cost if wrong: a stray hand-written series blocks backfill and delete mirroring until removed.
Ruling: implementer report files use .txt (task-N-report.txt) — implementers decline to write .md files under their own standing rules — cost if wrong: none.
Task 1: implementer DONE — GO (list form 168.6 ms Grafana / 30 ms engine at 3,070,727 series; .+ 158.5 / 29 ms); evidence commit e3fcafaa1 on bench branch; VM PMM-15663-bench2 torn down; in review
Task 3: minor (deferred): no test that the password never appears in errors (closed server / cancelled ctx)
Task 3: minor (deferred): Keys aborts on one bad series — covered by the Keys ruling above
Task 3: minor (deferred): Event.timeSeries() lives in store.go
Task 3: minor (deferred): `do` errors don't name the operation; error body read unbounded
Task 3: note for Task 7: Write sends one request — cap the batch size in the sync; a key missing from VictoriaMetrics must only ever cause a rewrite, never a delete
Task 3: complete (commits 9702d85c3..c80f2b67a, review clean)
Task 4: dispatched (agent a43d850, BASE c80f2b67a)
Task 1: minor (deferred): RESULTS.md P5 says the list form costs "about 10 ms more" — within run-to-run spread
Task 1: minor (deferred): run order after ingest not recorded; report says "uncommitted" (stale)
Task 1: note for Task 12 HA: expression size scales with name length (10 KB for 1,000 9-char names); HA vmselect may keep VictoriaMetrics' default -search.maxQueryLen (16 KB per the VM research, unverified) — check node overviews with long names in the HA run; PR 6045's HA run passed large panel queries, so the limit may already be raised
Task 1: complete (bench commit e3fcafaa1, review clean; script preserved as dev/pmm-15663-bench/bench_node_list.py)
Task 4: implementer DONE_WITH_CONCERNS, commit 8e5bdc619 (24 files incl. generated)
Ruling: generated files stay exactly as the container's `make gen` writes them for the files a task changes (annotation.pb.go lacks the one blank line every other pb.go has; the controller confirmed GOTOOLCHAIN=go1.27.1 gives the same), unrelated churn is reverted, nothing is hand-edited — CI's `make gen` + `git diff --exit-code` on the PR decides, and its log shows the exact diff — cost if wrong: one CI round to regenerate.
Task 4: pre-existing failure, not this change: TestAgentService/ListAgents/should_output_a_list_of_agents_provisioned_by_default fails identically on the base tree in the dev container
Task 4: review — spec ✅, 1 Important (annotation.pb.go lost the blank line before `var file_..._depIdxs`), 6 Minor
Ruling (supersedes the generated-files ruling above): the container's generator drops a blank line that CI's keeps — merged PR d61265739 regenerated api/inventory/v1/agents.pb.go with the line and passed CI's make gen + git diff gate — so a task's own regenerated *.pb.go gets that one blank line restored to equal CI's output — cost if wrong: one CI round.
Task 4: minor (deferred): zero-id guard — CreateAnnotation returns 0, nil if Grafana's 2xx lacks an id (final review: likely worth fixing, Task 5 deletes by id)
Task 4: minor (deferred): validation test covers one object only; no comment on validate-before-id relying on "0" being non-empty; no server log on the Unavailable path; plain errors for label-merge failures; annotationStore not alphabetical in deps.go
Task 4: fix round 1/5 dispatched — restore blank line in annotation.pb.go (commit a520f3a2c); re-review running
Task 4: fix round 1/5 (1 addressed, 0 open — blank line restored in annotation.pb.go; commits 8e5bdc619..a520f3a2c)
Task 4: complete (commits c80f2b67a..a520f3a2c, review clean after 1 fix round)
Task 5: dispatched (BASE a520f3a2c)
Ruling: Task 10 runs in parallel with Task 5 — separate repository, no shared files — cost if wrong: none.
Task 5: dispatched (agent ad684c3, BASE a520f3a2c)
Task 10: dispatched (agent aea2e01, percona-helm-charts branch PMM-15663-vmauth-delete-route from origin/main 2af1f99)
Task 10: implementer DONE_WITH_CONCERNS, chart commit 98ad2a1 (pushed); helm template needed a scratch copy without pg-user-credentials-secrets.yaml (cluster lookup)
Ruling: the HA delete route uses the same vmauth user as writes and reads, so any holder of that credential (PMM Server components, admins via nginx) can delete any series — same reach as a single-node admin via /prometheus — accepted for this ticket and raised to the user as a hardening option (restrict the route, e.g. vmauth src_query_args on match[]) — cost if wrong: a leaked vmauth credential can delete metrics, not only write them.
Ruling: percona-helm-charts refuses pushes from this session (Claude GitHub App not installed for it); the chart commit is kept as a patch on the bench branch (dev/pmm-15663-bench/helm/) until the user grants access or pushes it — cost if wrong: none, the patch applies with git am.
Task 10: minor (deferred): shared vmauth credential can now delete — covered by the security ruling above (Low per reviewer; follow-up: separate vmauth user for deletes)
Task 10: complete (chart commits 2af1f99..98ad2a1, review clean; patch on bench branch 7ca98737b, not pushed to percona-helm-charts — no access)
Task 5: implementer DONE_WITH_CONCERNS, commit e063b3314 (extracted callerAuthorization helper shared with AddAnnotation)
Task 5: pre-existing: `go test ./services/management/...` in parallel hits a duplicate testdb name in backup; serial run (-p 1) clean
