# Kernel compatibility matrix (Phase 1 hardening)

Representative targets for CI / manual validation (extend with your fleet images):

| Environment | Kernel | Notes |
|-------------|--------|-------|
| RHEL / OL 8/9 LTS | 4.18+ / 5.14+ | Verify BTF (`/sys/kernel/btf/vmlinux`) for CO-RE when probes ship |
| Ubuntu 22.04 / 24.04 LTS | 5.15 / 6.8 | cgroup v2 hybrid paths |
| EKS / GKE / AKS node images | vendor track | Confirm SYS_ADMIN / CAP_BPF / CAP_PERMON policy for DaemonSet |

**Preflight (future agent checks):** kernel version, BTF presence, mount namespaces, attach errors surfaced as `degraded_reason`.
