# coroot-node-agent on Kubernetes (reference)

Step-by-step test flow (OTLP to PMM, `/metrics` for vmagent, stash vs kept files): **`docs/internal/2026-04-08_coroot-k8s-pmm-test-runbook.md`**.

PMM **v1** expects the **PMM Helm chart** to own DaemonSet install/upgrade/delete for `coroot-node-agent`. **`daemonset.example.yaml`** uses the **official** image **`ghcr.io/coroot/coroot-node-agent`** (upstream entrypoint: `coroot-node-agent`). If you prefer the binary shipped inside **pmm-client** instead, replace the image and set `command` to `/usr/local/percona/pmm/tools/coroot-node-agent` on a build with **`WITH_COROOT_AGENT=1`**.

For **node/bare-metal** installs, **`pmm-admin add ebpf`** merges eBPF-related labels on the node’s **`otel_collector`** inventory row (same as other `pmm-admin add otel` subcommands). Remove or adjust that agent via the Inventory API or `pmm-admin inventory remove agent <id>` as appropriate for your version.

On Kubernetes, run **coroot-node-agent** as a DaemonSet (see the runbook linked above); there is typically **no** local `pmm-admin` on the node unless you also run **pmm-agent** there.

## Reference manifest

`daemonset.example.yaml` is an **operator reference** aligned with Pod Security **baseline**-class ideas: adjust `securityContext`, capabilities, and volume mounts to match your cluster policy and the upstream coroot-node-agent requirements for your kernel.

Before applying:

1. Pin **`ghcr.io/coroot/coroot-node-agent:<version>`** to a release you validate (see [GHCR packages](https://github.com/coroot/coroot-node-agent/pkgs/container/coroot-node-agent)).
2. Set PMM Server OTLP/registration settings consistent with other pmm-agent installs.
3. Review capabilities (`CAP_BPF`, `CAP_PERFMON`, `CAP_SYS_ADMIN`, etc.) against upstream coroot-node-agent requirements for your kernel and cluster policy.

The authoritative chart integration lives in the **PMM Helm chart repository** (not this file); keep this directory in sync with chart changes when possible.
