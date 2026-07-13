# Runbook: eBPF / OTLP collector canary rollout

## Preconditions

- PMM server OTEL collector enabled; `/srv/otelcol/config.yaml` managed by `pmm-managed`.
- ClickHouse schemas applied (`otel.logs`, `otel.otel_traces`, `otel.otel_metrics_sum`, `otel.service_map_*`).

## Canary steps

1. **Enable on one host** — `pmm-admin management add otel ebpf` (or `add otel traces` / `add otel logs` as needed). Confirm agent connects and no sustained scrape errors.
2. **Stub validation** — from a workstation with CA trust, run `go run ./managed/cmd/ebpf-otlp-stub` with `PMM_OTLP_URL` pointing at `https://<pmm>/otlp/v1/traces` and `PMM_OTLP_INSECURE=1` only on lab systems. Verify rows in `otel.otel_traces`.
3. **SLO spot-check** — watch collector queue depth / dropped batches; target **&lt; 1%** drops over 30 minutes at expected QPS.
4. **Expand** — repeat per cluster / node pool; document kernel version and BTF availability per image.

## Rollback

- `pmm-admin inventory remove-agent <id>` (or management remove flow) for the OTEL collector agent; server-side OTEL continues to serve other agents.

## Kernel / caps failures

- Agent should stay healthy with **degraded** eBPF state once probes exist; OTLP-only path remains for logs/traces/metrics from other receivers.
