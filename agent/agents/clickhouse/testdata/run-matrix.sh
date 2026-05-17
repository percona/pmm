#!/usr/bin/env bash
# Runs the ClickHouse collector integration tests across the full support
# matrix: every supported version × {single-node, cluster} topology.
#
# Each combination is started, validated and torn down in turn, so the matrix
# fits within modest memory. Override the matrix with env vars:
#
#   CLICKHOUSE_VERSIONS="25.3 24.8"  CLICKHOUSE_TOPOLOGIES="single" ./run-matrix.sh
#
# Usage:  bash run-matrix.sh
set -uo pipefail

cd "$(dirname "$0")"
repo_root=$(cd ../../../.. && pwd)
compose=(docker compose -f docker-compose.matrix.yml)

# Build the clickhouse_exporter binary once, up front, so the exporter matrix
# test (TestClickHouseExporterMatrix) can launch the packaged-equivalent binary
# per endpoint. Its path is handed to the tests via CLICKHOUSE_EXPORTER_BIN.
exporter_bin="${repo_root}/bin/clickhouse_exporter"
echo ">>> building clickhouse_exporter"
if ! CGO_ENABLED=0 go -C "$repo_root" build -o bin/clickhouse_exporter ./agent/cmd/clickhouse_exporter; then
    echo "!!! failed to build clickhouse_exporter"
    exit 1
fi
export CLICKHOUSE_EXPORTER_BIN="$exporter_bin"

# Supported ClickHouse versions and topologies (override via env).
read -r -a versions <<<"${CLICKHOUSE_VERSIONS:-26.3 25.8 25.3 24.8 24.3}"
read -r -a topologies <<<"${CLICKHOUSE_TOPOLOGIES:-single cluster}"

rc=0
for v in "${versions[@]}"; do
    for topo in "${topologies[@]}"; do
        echo ">>> ClickHouse ${v} / ${topo}"
        export CLICKHOUSE_IMAGE="clickhouse/clickhouse-server:${v}"

        if ! "${compose[@]}" --profile "$topo" up -d --wait --wait-timeout 240; then
            echo "!!! ${v}/${topo}: containers did not become healthy"
            "${compose[@]}" --profile "$topo" logs --tail 30 || true
            "${compose[@]}" --profile "$topo" down -v >/dev/null 2>&1 || true
            rc=1
            continue
        fi

        if [ "$topo" = "single" ]; then
            endpoints="single-${v}=clickhouse://default:clickhouse@127.0.0.1:9000/default"
        else
            endpoints="cluster-${v}-node1=clickhouse://default:clickhouse@127.0.0.1:9100/default"
            endpoints+=",cluster-${v}-node2=clickhouse://default:clickhouse@127.0.0.1:9101/default"
        fi

        # The filter intentionally also matches TestClickHouseExporterMatrix and
        # TestClickHouseQANMatrix. Those integration tests are not committed yet;
        # `go test -run` simply skips names it cannot find, so the matrix stays
        # runnable today and picks them up automatically once they land.
        # TODO(clickhouse-epic): add exporter_integration_test.go and
        # querylog/qan_integration_test.go (build tag clickhouse_integration).
        if ! CLICKHOUSE_TEST_ENDPOINTS="$endpoints" \
            go -C "$repo_root" test -tags clickhouse_integration -count=1 -v \
            -run 'TestClickHouse(Matrix|ExporterMatrix|QANMatrix)' ./agent/agents/clickhouse/...; then
            echo "!!! ${v}/${topo}: integration tests failed"
            rc=1
        fi

        "${compose[@]}" --profile "$topo" down -v >/dev/null 2>&1 || true
    done
done

if [ "$rc" -eq 0 ]; then
    echo "=== ClickHouse matrix: ALL PASSED ==="
else
    echo "=== ClickHouse matrix: FAILURES (see above) ==="
fi
exit "$rc"
