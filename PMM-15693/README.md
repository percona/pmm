# PMM-15693 live verification evidence

Evidence for the PMM-15693 fix PR (branch `PMM-15693-ha-follower-agent-state`). Not for merge; delete this branch after the PR merges.

- Cluster: CHAOS k3s, 3 nodes, chart `pmm-ha` 1.8.0 (PMM-HA-GA), image `perconalab/pmm-server:3-dev-latest` (pmm-managed `a27167e1`, the PR's base commit).
- Fix: pmm-managed built from `78494843` (sha256 `e246bf91…`), hot-swapped into all three pods.
- `before-step1.txt`, `before-step2.txt`: the bug on the stock binary.
- `after-step1.txt`, `after-step1-logs.txt`, `after-idle.txt`, `after-step2.txt`: the same steps with the fix.
- `PMM-15693-explore.png`: Grafana Explore, hr scrape samples per minute per replica (12 = 5s, 6 = 10s), 20:36Z–20:50Z.
