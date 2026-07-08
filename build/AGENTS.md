# PMM Build and Packaging Guidelines

> **Parent guide**: [AGENTS.md](../AGENTS.md) — product overview, architecture, domain model, global conventions

The `/build` directory contains everything needed to build, package, and distribute PMM Server and PMM Client as Docker images, RPM/DEB packages, and cloud machine images (AMI).

## Architecture

### Build Artifacts

| Artifact | Format | Source |
|----------|--------|--------|
| PMM Server Docker image | Docker (EL9) | `docker/server/Dockerfile.el9` |
| PMM Client Docker image | Docker (EL9) | `docker/client/Dockerfile.el9` |
| PMM Server RPMs | RPM (EL9) | `packages/rpm/server/SPECS/` |
| PMM Client RPM | RPM (EL9) | `packages/rpm/client/pmm-client.spec` |
| PMM Client DEB | DEB | `packages/deb/` |
| PMM Server AMI | AWS AMI | `packer/pmm.json` |

### Build Pipeline

```
Source code (Go, TypeScript)
  → Binary compilation (make release)
    → RPM/DEB packaging (packages/)
      → Docker image build (docker/)
        → Machine image build (packer/)
```

## Ansible Roles

### Server Provisioning Roles

| Role | Purpose |
|------|---------|
| `clickhouse` | Install and configure ClickHouse for QAN |
| `grafana` | Install Grafana, provision datasources and dashboards |
| `nginx` | Configure Nginx as reverse proxy (SSL termination, routing) |
| `postgres` | Install and configure PostgreSQL for pmm-managed |
| `supervisord` | Configure Supervisord for process management |
| `dashboards` | Provision PMM Grafana dashboards |
| `initialization` | PMM Server first-run setup |
| `pmm-images` | Image metadata and version info |

### Cloud-Specific Roles

| Role | Purpose |
|------|---------|
| `cloud-node` | Cloud VM preparation |
| `lvm-init` | LVM storage setup for persistent data |
| `init-admin-password-ami` | Set admin password from EC2 instance ID |
| `ami` | AMI-specific customization |

## Packer Templates

### PMM Server Images (`packer/pmm.json`)

Builds machine images:
- **amazon-ebs** — AWS AMI

### Make Targets

```bash
make pmm-ami              # Build AWS AMI
make rpmbuild-el9         # Build RPM build environment image
```

## Patterns and Conventions

### Do
- Use Ansible roles for server provisioning — they're the single source of truth for PMM Server setup
- Keep Dockerfiles minimal — delegate to Ansible for complex provisioning
- Use multi-stage Docker builds where appropriate
- Keep RPM/DEB specs in sync with actual binary and config file paths
- Test image builds in CI before merging

### Don't
- Don't hardcode versions in Dockerfiles — use build args
- Don't modify Ansible roles without testing the full image build
- Don't add secrets or credentials to build scripts or Dockerfiles
- Don't skip the RPM build step when modifying server components

## Key Files to Reference

- `build/docker/server/Dockerfile.el9` — PMM Server Docker image definition
- `build/docker/server/entrypoint.sh` — Server container entrypoint
- `build/ansible/pmm-docker/main.yml` — Docker provisioning playbook
- `build/ansible/roles/` — All Ansible roles for server components
- `build/packages/rpm/server/SPECS/` — RPM spec files for server components
- `build/packer/pmm.json` — Machine image definitions
- `build/scripts/` — Build scripts for all artifact types
