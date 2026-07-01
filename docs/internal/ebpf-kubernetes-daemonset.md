# Kubernetes DaemonSet notes (Phase 1 eBPF collector)

Use the same `pmm-client` image as bare metal. PMM agent registers with the server; `otelcol-contrib` exports OTLP to PMM.

## Security context (indicative)

- Capabilities: typically `CAP_BPF`, `CAP_PERFMON`, `CAP_SYS_ADMIN` (trim to least privilege validated on your kernel).
- `hostPID`: often required for meaningful host-wide capture (validate against threat model).
- Volume mounts: `/sys` / `debugfs` / BTF paths as required by CO-RE probes when enabled.

## Resources

- Set CPU/memory requests and limits; eBPF map and ring-buffer sizes must fit within the cgroup.

## Priority

- Consider `priorityClassName` so the DaemonSet is not evicted before workloads on saturated nodes.

See also `docs/runbooks/ebpf-rollout-canary.md`.
