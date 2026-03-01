# Migration plan - PMM Server direct binary builds

## Plan: Migrate PMM Server from RPM to direct binary builds
Simplify the build pipeline by eliminating RPM packaging for all server components using Docker multi-stage builds. Build Go binaries in golang:latest, Node.js artifacts in node:latest, then copy to Oracle Linux runtime with exact same paths as RPM specs defined.

## Steps
1. Add multi-architecture build support — Update all Makefiles to accept GOARCH parameter (defaulting to amd64). Ensure all release targets use GOOS=linux GOARCH=${GOARCH} for cross-compilation. Configure Docker buildx in build/bin/build-server-docker with --platform linux/amd64,linux/arm64 and enable BuildKit (DOCKER_BUILDKIT=1).

2. Create multi-stage Dockerfile — Restructure build/docker/Dockerfile.el9 with build stages: (1) golang:latest AS go-builder for pmm-managed, qan-api2, vmproxy, pmm-dump, VictoriaMetrics, and Grafana Go binaries using cache mounts --mount=type=cache,target=/go/pkg/mod, (2) node:latest AS node-builder for Grafana UI and PMM UI builds using cache mounts --mount=type=cache,target=/root/.npm, (3) node:latest AS dashboards-builder for percona-dashboards, (4) oraclelinux:9-slim AS runtime copies all artifacts with precise ownership and permissions.

3. Build all Go binaries in go-builder stage — In go-builder stage, clone and build pmm-managed (from managed), qan-api2 (from qan-api2), vmproxy (from vmproxy), pmm-dump (github.com/percona/pmm-dump), Grafana Go binaries (github.com/percona/grafana with make build-go), and VictoriaMetrics (github.com/VictoriaMetrics/VictoriaMetrics with make victoria-metrics-pure and make vmalert-pure). Respect TARGETARCH for multi-platform builds.

4. Build all Node.js artifacts in node-builder stages — In node-builder stage, clone and build Grafana UI (github.com/percona/grafana with npm install and make build-js) producing public/, conf/, tools. In dashboards-builder stage, clone and build percona-dashboards (github.com/percona/grafana-dashboards) and PMM UI (from ui with make release).

5. Install system dependencies in runtime stage — In runtime stage, install required system packages via microdnf install -y fontconfig ansible-core epel-release postgresql clickhouse-server nginx supervisord and other dependencies from current build/ansible/roles/pmm-server/tasks/main.yml before copying binaries.

6. Copy binaries to exact RPM paths with root:root ownership — In runtime stage, copy executable binaries from builder stages to exact paths defined in RPM specs: COPY --from=go-builder --chown=root:root --chmod=0755 /build/pmm-managed /usr/sbin/pmm-managed, /usr/sbin/pmm-encryption-rotation, /usr/sbin/pmm-managed-init, /usr/sbin/pmm-managed-starlark, /usr/sbin/percona-qan-api2, /usr/sbin/vmproxy, /usr/sbin/pmm-dump, /usr/sbin/grafana-server, /usr/sbin/grafana, /usr/bin/grafana-cli, /usr/sbin/victoriametrics, /usr/sbin/vmalert. Paths must match RPM spec %install sections exactly to ensure supervisord compatibility.

7. Copy assets to exact RPM paths with pmm:pmm ownership — Copy to exact paths from RPM specs: Grafana assets to /usr/share/grafana/public, /usr/share/grafana/conf, /usr/share/grafana/tools. Grafana configs to /etc/grafana/grafana.ini, /etc/grafana/ldap.toml. PMM UI to /usr/share/pmm-ui. Percona dashboards to /usr/share/percona-dashboards. pmm-managed assets to /usr/share/pmm-managed. All with --chown=pmm:pmm. Create directories with RUN mkdir -p /var/lib/grafana && chown pmm:pmm /var/lib/grafana.

8. Remove RPM infrastructure and update Ansible — Delete SPECS directory completely. Remove all build-server-rpm calls from build/bin/build-server-docker. In build/ansible/roles/pmm-server/tasks/main.yml, remove local yum repository setup and all server component dnf install tasks. Move system package installation to Dockerfile runtime stage before Ansible execution.

9. Create VERSION.json for introspection — In runtime stage after copying binaries, generate /usr/share/pmm-server/VERSION.json using build args: ARG PMM_REF, ARG GRAFANA_COMMIT, ARG VM_VERSION, ARG DASHBOARDS_COMMIT, ARG PMM_DUMP_COMMIT. Run RUN echo '{"pmm-managed": "'${PMM_REF}'", "qan-api2": "'${PMM_REF}'", "vmproxy": "'${PMM_REF}'", "grafana": "'${GRAFANA_COMMIT}'", "victoriametrics": "'${VM_VERSION}'", "percona-dashboards": "'${DASHBOARDS_COMMIT}'", "pmm-dump": "'${PMM_DUMP_COMMIT}'"}' > /usr/share/pmm-server/VERSION.json && chmod 0644 /usr/share/pmm-server/VERSION.json.

## Further Considerations
1. Document exact RPM path mappings for all components? — Create a reference mapping between each RPM spec file's %install and %files sections to ensure all artifacts are copied to identical paths in the Dockerfile. Review all server spec files before deletion.

2. CI/CD pipeline updates needed? — Build scripts that reference RPM artifacts or use RPM commands for verification need updating. Check Jenkins/GitHub Actions workflows for RPM-specific logic that should be removed or replaced.
