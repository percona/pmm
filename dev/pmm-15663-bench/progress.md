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
Ruling: api/descriptor.bin stays untouched on this branch — CI only logs the `make -C api descriptors` diff (main.yml:47-52), the change is additive, and keeping the baseline leaves the unreleased RPC free to change — cost if wrong: the descriptor gets refreshed in a later PR or at release.
Ruling: RemoveAnnotation returns Unauthenticated without an Authorization header, exactly as AddAnnotation does (brief-mandated) — cost if wrong: a cookie-only UI caller gets 401; noted for live QA and PMM-15073.
Task 5: minor (deferred): non-404 Grafana errors pass through as codes.Unknown with the Grafana URL/body (inherited from AddAnnotation)
Task 5: minor (deferred): Remove subtests build a testdb they don't need; DeleteAnnotation doc comment is two sentences; proto comment wording "is a params"
Task 5: complete (commits a520f3a2c..e063b3314, review clean)
Task 6: dispatched (BASE e063b3314)
Ruling: pmm-admin annotationResult.AnnotationID is a string — the generated client types int64 as string (grpc-gateway JSON) — test uses "17", output unchanged — cost if wrong: none.
Task 6: implementer DONE, commit 55e1ea0d4
Task 6: review — spec ✅; reviewer's Critical (Co-authored-by trailer) overruled by the trailer ruling above (runner-hook policy, already on pushed commits); Important: --json key must be snake_case annotation_id
Task 6: minor (deferred): nil payload prints "Annotation ID: " with empty value; RunCmd extraction not unit-tested; resp vs result naming
Task 6: fix round 1/5 dispatched (json tag)
Task 6: fix round 1 commit 4a3257c45 (pushed), re-review running
Ruling: Task 7 dispatched before Task 6's re-review returns — the re-review is read-only on a one-line fix, and any further Task 6 fix waits until Task 7 finishes so two agents never commit in the worktree at once — cost if wrong: a Task 6 fix lands after Task 7's commits.
Task 7: dispatched (agent aa41cbe, model opus, BASE 4a3257c45, decisions in task-7-decisions.txt)
Task 6: fix round 1/5 (1 addressed, 0 open — snake_case json tag; commits 55e1ea0d4..4a3257c45)
Task 6: minor (deferred): fix commit title is 51 chars (rule ≤50); squash-merge uses the PR title, so it does not reach main
Task 6: complete (commits e063b3314..4a3257c45, review clean after 1 fix round)
Ruling: Task 9 runs in parallel with Task 7 in a second worktree (.superpowers/wt-dash, branch PMM-15663-dashboards-wip from 4a3257c45) — it touches only dashboards/, Task 7 only managed/ — its commits are cherry-picked onto PMM-15663-scoped-annotations afterwards — cost if wrong: a cherry-pick conflict, resolved by hand.
Ruling: the new dashboard entry uses the same Prometheus datasource reference as the dashboard's own panels (mostly the name "Metrics"), not the benchmark's uid — consistent with every panel — cost if wrong: none, Grafana resolves both.
Ruling: dashboard matchers include only labels the term's series carry (node term: node_name/node_name_list, region, az, node_type) — a node series has no environment/cluster — cost if wrong: none.
Task 9: dispatched (agent a782aba, model opus, worktree wt-dash, BASE 4a3257c45)
Task 9: implementer DONE_WITH_CONCERNS, commit c825cb592 on PMM-15663-dashboards-wip (61 dashboards: 39 service, 14 node, 8 both; 7 new hidden node_name_list)
Ruling: D1 accepted — matchers are the labels each dashboard's panels filter on plus the term's own node/service, not every fleet variable; spec 5.5 says "Matchers copy each dashboard's panel matchers", and hidden All-without-allValue variables would otherwise hide annotations the panels show — cost if wrong: an unfiltered fleet variable doesn't narrow annotations.
Ruling: D2 (node_name_list whenever node_name can be All with an allValue), D3 (enable/hide/iconColor copied from the sibling "PMM Annotations" entry) and D5 (test_all_value.py counts annotation use of a _list variable as used) accepted — consistent with the spec, reviewed in the task review — cost if wrong: small, reviewer checks.
Task 7: implementer DONE_WITH_CONCERNS, commits 678e8f458, f4d1c295e, e288a7680; TestImports fails since Tasks 4–5 (management imports services/annotations and services/grafana)
Ruling: ErrAnnotationNotFound moves to the shared package managed/services (errors.go, where other sentinels live and which both grafana and management may import); TestImports whitelists services/annotations for management (same mechanism inventory uses for management/common) and lists annotations among the independent services; packages.dot regenerated — fixes a Task 4–5 defect, done by the Task 7 implementer before review — cost if wrong: a reviewer may prefer moving annotations out of services/.
Ruling: a tag naming both a node and a service with no text suffix naming it yields both events (as Grafana's tag match would show it) — cost if wrong: one extra marker on the other dashboard type.
Ruling: accepted risk — if the Grafana DSN pointed at a different database that has an `annotation` table, the delete step would remove most series; it is the DSN the alerting provisioner already trusts — cost if wrong: series deleted until the next backfill rewrites them from the real Grafana DB (only annotations within retention).
Task 7: TestImports fix done — commits 7e6757af4, 809191646; reviewing range 4a3257c45..809191646
Ruling: Task 11 runs in parallel in a third worktree (.superpowers/wt-docs, branch PMM-15663-docs-wip from 809191646), touching only documentation/; its `make doc-build` check moves to Task 12, when the shared dev VM is free — commits cherry-picked afterwards — cost if wrong: a docs build error found later.
Ruling: no new release-notes file — the next version file does not exist and the target release is undecided; the release-note entry is drafted in the Task 11 report for the PR description — cost if wrong: the release owner adds it to the version file.
Task 11: dispatched (agent a0935a1, worktree wt-docs)
Task 9: review — ❌ 2 Important: (1) service term includes child node_name derived from $service_name (MySQL_Table_Details, MySQL_Replication_Summary); (2) MongoDB_ReplSet_Summary should use $service_name_list (set narrowing); D4, D5 accepted by reviewer
Ruling: pmm-qan stays rewritten — consistent with the scope mapping; its entry is hidden — cost if wrong: an unneeded node_name_list refresh on QAN.
Ruling: both-scope overviews keep node annotations unnarrowed by the service selection — unchanged mapping from main (spec 5.5) — cost if wrong: node annotations of other nodes in node_name_list show when one service is picked.
Task 9: minor (deferred): no expected-scope map in tests (a dropped term would pass); is_prometheus accepts any non "-- " string; VictoriaMetrics_Agents_Overview node term uses $node_name not $node_name_list; commit title 54 chars (brief-prescribed)
Task 9: fix round 1/5 dispatched (2 findings)
Task 11: implementer DONE_WITH_CONCERNS, commit bf9c9d6a1 on PMM-15663-docs-wip (annotate.md, pmm-admin config.md Output section, ui_components.md)
Ruling: user docs say LBAC applies to scoped annotations on dashboards and add no caveat about Grafana's own annotations API (which ignores LBAC for every annotation, before and after this change); the gap goes to the user in the final report — cost if wrong: docs silent on a pre-existing limitation.
Task 11: note for Task 12: retake PMM_Common_Controls_Annotation_Toggle.jpg (now two switches) during live verification
Task 7: minor (deferred, final review should triage): grafanaDB keeps database/sql pool defaults (alerting reader uses 1 conn, 1 min lifetime) — stale-connection error after idle in HA
Task 7: minor (deferred, final review should triage): first pass on a fresh install logs Error before Grafana tables exist (42P01); alerting treats it as expected
Task 7: minor (deferred, final review should triage): mass-delete guard — existence query returning 0 of N>0 ids deletes all; a "0 of N: warn and skip" guard would cover the wrong-DSN ruling too
Task 7: minor (deferred, final review should triage): DSN parse error may log a GF_DATABASE_URL password (pre-existing in alerting, now logged twice)
Task 7: minor (deferred): PMM DB reads ignore the pass ctx; repeated Validate warning every pass; tag order without ORDER BY; test builds mockSyncStore by hand
Task 7: complete (commits 4a3257c45..809191646, review clean)
Task 8: dispatched (BASE 809191646)
Task 9: fix round 1 commit 98dbd022c (20 dashboards: derived node_name dropped from service term; $service_name_list where panels use it); re-review running
Task 11: review — ❌ 4 Important (release note says turn on a default-on switch; selection claim too broad on both-scope dashboards; marker title claimed for global annotations; promises editing)
Task 11: minor (deferred): Editors vs Grafana API delete wording; label count includes __name__; link texts vs target H1s; release-note bullet long; HA 90-day retention; POST Unavailable retry creates a duplicate; curl without -k
Task 11: fix round 1/5 dispatched (4 findings)
Task 11: fix round 1 commit d16249691 (4 addressed per implementer + optional custom-labels wording); re-review running
Task 9: fix round 1/5 (2 addressed, 0 open — derived node_name dropped from service terms on 20 dashboards, $service_name_list where panels use it; commits c825cb592..98dbd022c)
Task 9: minor (deferred): descendants() duplicated in tool and test (test checks consistency, not independent derivation)
Task 9: complete (commits 4a3257c45..98dbd022c on PMM-15663-dashboards-wip, review clean after 1 fix round); cherry-pick c825cb592 98dbd022c after Task 8
Task 11: fix round 1/5 (4 addressed, 0 open; commits bf9c9d6a1..d16249691)
Task 11: minor (deferred, final review should check against the merged dashboards): annotate.md:26 "a dashboard with both node and service variables shows both" — scope follows the old tags, not the variables (e.g. MySQL_Amazon_Aurora_Details has a hidden node_name but is service-only); page doesn't say the new switch is on by default; label count omits __name__
Task 11: complete (commits 809191646..d16249691 on PMM-15663-docs-wip, review clean after 1 fix round); cherry-pick bf9c9d6a1 d16249691 after Task 8
Task 8: implementer DONE_WITH_CONCERNS, commit 572d44091; live server showed Grafana 12.4.5 answers 403 (not 404) for DELETE of an unknown annotation id -> RemoveAnnotation returned 500
Ruling: Grafana's 403 on DELETE /api/annotations/{id} maps to services.ErrAnnotationNotFound like 404 — only PMM admins reach RemoveAnnotation and a Grafana admin always holds annotations:delete, so 403 there means an unresolvable id — fixed by the Task 8 implementer (it holds the live setup) — cost if wrong: a genuine Grafana permission failure is reported as NotFound.
Ruling: the api-test checks NotFound with a never-existing id instead of "remove again" — Grafana keeps answering 200 for a just-deleted id for ~20–50 s — cost if wrong: none.
Task 8: 403 fix commit 4f74d9a07; live: unknown id -> 404, TestAddAnnotation (8) and TestRemoveAnnotation (2) pass against the hot-swapped server; in review
Cherry-picked Task 9 (c825cb592 -> e0bf8940b, 98dbd022c -> 83ef785fd) and Task 11 (bf9c9d6a1 -> 3e6761316, d16249691 -> 93eabfcc6) onto PMM-15663-scoped-annotations; dashboard suite green on the combined branch; pushed
Task 8: minor (deferred, final review should triage): literal "annotation-node"/"annotation-service" names now carry series assertions on a shared server (use TestString); "Add Basic Annotation" leaks a global annotation per run; annotationSeries fails on a transient VM error instead of retrying; 403 comment lacks the why; stale commit body
Task 8: complete (commits 809191646..4f74d9a07, review clean)
Final review: dispatched (agent abb4cd2, opus, range be751c4b6..93eabfcc6, triage list final-triage-list.txt)
Final review: With fixes — 0 Critical, 4 Important (docs in code PR; label-value length; reserved-label LBAC collision; Task 12 open), must-fix minors (DSN password in logs; docs: Editor wording, both-variables sentence, "and no others")
Ruling: docs leave the code PR — reverted on PMM-15663-scoped-annotations (4ca956d86, 781472618; a revert rather than a force-push of the pushed branch); PMM-15663-docs-wip becomes the separate docs PR, merged when the release ships (documentation/AGENTS.md: merge to main publishes) — cost if wrong: none.
Ruling: reserved series labels are renamed annotation_id/text/tags -> pmm_annotation_id/pmm_annotation_text/pmm_annotation_tags — a custom label named tags/text/annotation_id would otherwise break LBAC (pass/fail requirement); rejecting collisions would stop scoped annotations for objects with a plausible custom label like "tags"; nothing has shipped — cost if wrong: one more rename later.
Ruling: Validate rejects any series label value over MaxTextBytes (after text truncation) with a new sentinel, mapped to InvalidArgument — VictoriaMetrics drops such series silently (probe P1) — cost if wrong: none.
Ruling: fix wave also takes cheap items — Keys read before Grafana in syncOnce (removes the Remove/sync race); RemoveAnnotation still deletes the series when Grafana says not found, then returns NotFound; Grafana DB pool like the alerting reader; ORDER BY in the Grafana read; pmm-admin omits an empty id line; managed/AGENTS.md lists services/annotations and the VictoriaMetrics write path — cost if wrong: small.
Ruling: parked (can wait, per final triage) — sync taking scope from the text suffix (user tag equal to a node/service name; names with 2+ colons), Keys strict, leader-only registration unit test, error-log rate on a permanent delete failure, PMM DB reads without ctx, and every other deferred minor the final triage marked can-wait — cost if wrong: edge-case duplicate markers; colon-heavy names not backfilled.
Ruling: DESIGN §6 corrected by the controller — Editors can still delete scoped annotations through Grafana's annotations API (the sync job mirrors it); only the PMM API is admin-only — cost if wrong: none.
Final fix wave: DONE_WITH_CONCERNS — code 4ca956d86..41c6cff58 (10 commits), docs d16249691..d16104201; VERIFY passed incl. live api-tests; re-review next
Final fix wave: re-review — all 14 items (F1–F9, D1–D5) ADDRESSED, no new breakage (code 4ca956d86..41c6cff58, docs d16249691..d16104201)
Final: parked — Ruling: a pmm_annotation_event series without pmm_annotation_id (written only by pre-F1 builds of this unreleased branch) makes Keys fail every sync pass until it leaves retention; affects only servers that ran an intermediate build (none released); a `{pmm_annotation_id!=""}` filter in Keys would avoid it — cost if wrong: a QA server that ran an early build needs its old series deleted by hand.
Final: parked — Ruling: the docs PR does not yet mention the 4,096-byte value rejection, and config.md says the ID is always printed (empty id line omitted against an older server) — the docs PR merges at release and can take both lines then — cost if wrong: two doc sentences.
Final: controller updated DESIGN §5.3 (403 → NotFound, series delete still runs) and §5.4 (Keys read before Grafana) to match F4/F5.
Final review: clean after one fix wave. Task 12 (Feature Build, live single-node and HA verification, make prepare-pr, make doc-build, screenshot) not started — needs the user's go-ahead (Feature Build PR, chart push access).
