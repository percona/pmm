---
applyTo: build/**
---
# PMM Build and Packaging Guidelines

> **Parent guide**: [PMM_AGENTS.md](../../PMM_AGENTS.md) — product overview, architecture, domain model, global conventions

The `/build` directory contains everything needed to build, package, and distribute PMM Server and PMM Client as Docker images, RPM/DEB packages, and cloud machine images (AMI, OVA, Azure, DigitalOcean).

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
| PMM Server OVA | VirtualBox OVF | `packer/pmm.json` |
| PMM Server Azure | Azure image | `packer/pmm.json` |

### Build Pipeline

```
Source code (Go, TypeScript)
  → Binary compilation (make release)
    → RPM/DEB packaging (packages/)
      → Docker image build (docker/)
        → Machine image build (packer/)
```

## Directory Structure

```
build/
├── Makefile                         # Image build targets
├── ansible/
│   ├── ansible.cfg
│   ├── hosts                        # Inventory file
│   ├── pmm-docker/
│   │   └── main.yml                 # Docker image provisioning playbook
│   └── roles/
│       ├── clickhouse/              # ClickHouse installation and config
│       ├── cloud-node/              # Cloud-specific node setup
│       ├── dashboards/              # Grafana dashboard provisioning
│       ├── grafana/                 # Grafana installation and config
│       ├── init-admin-password-ami/ # AMI admin password initialization
│       ├── initialization/          # PMM Server first-run initialization
│       ├── lvm-init/                # LVM storage setup for cloud images
│       ├── nginx/                   # Nginx reverse proxy config
│       ├── pmm-client/              # PMM Client installation
│       ├── pmm-images/              # Image metadata
│       ├── postgres/                # PostgreSQL installation and config
│       └── supervisord/             # Supervisord process management config
├── docker/
│   ├── server/
│   │   ├── Dockerfile.el9           # PMM Server Docker image
│   │   ├── entrypoint.sh            # Container entrypoint
│   │   └── README.md
│   ├── client/
│   │   └── Dockerfile.el9           # PMM Client Docker image
│   └── rpmbuild/
│       ├── Dockerfile.el8           # RPM build environment (EL8)
│       ├── Dockerfile.el9           # RPM build environment (EL9)
│       └── Dockerfile.hetzner-el9   # Hetzner ARM build environment
├── docs/
│   ├── README.md                    # Build overview
│   ├── MIGRATION.md                 # Migration documentation
│   └── RELEASE_CANDIDATE.md         # Release process
├── packages/
│   ├── deb/                         # DEB packaging files
│   │   ├── control                  # Package metadata
│   │   ├── rules                    # Build rules
│   │   ├── preinst                  # Pre-install script
│   │   ├── postrm                   # Post-remove script
│   │   └── links                    # Symlinks
│   └── rpm/
│       ├── client/
│       │   └── pmm-client.spec      # PMM Client RPM spec
│       └── server/
│           └── SPECS/
│               ├── pmm-managed.spec
│               ├── grafana.spec
│               ├── victoriametrics.spec
│               └── ...              # Other server component specs
├── packer/
│   ├── pmm.json                     # PMM Server image: AMI, OVA, Azure, DigitalOcean
│   ├── aws.pkr.hcl                  # CI agent images (AWS)
│   ├── do.pkr.hcl                   # CI agent images (DigitalOcean)
│   ├── README.md
│   └── ansible/
│       └── roles/                   # Packer-specific Ansible roles
│           ├── ami-ovf/             # AMI/OVF specific setup
│           ├── cloud-node/          # Cloud node preparation
│           ├── lvm-init/            # LVM for cloud images
│           └── podman-setup/        # Podman container runtime
└── scripts/
    ├── build-client                 # Build PMM Client binaries
    ├── build-server                 # Build PMM Server binaries
    ├── build-client-docker          # Build PMM Client Docker image
    ├── build-server-docker          # Build PMM Server Docker image
    ├── build-client-deb             # Build PMM Client DEB package
    ├── build-server-rpm             # Build PMM Server RPMs
    └── install_tarball              # Install from tarball
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
| `ami-ovf` | AMI/OVF-specific customization |

## Packer Templates

### PMM Server Images (`packer/pmm.json`)

Builds machine images for multiple platforms:
- **amazon-ebs** — AWS AMI
- **azure-arm** — Azure managed image
- **virtualbox-ovf** — OVA for on-premises
- **digitalocean** — DigitalOcean snapshot

### Make Targets

```bash
make pmm-ami              # Build AWS AMI
make pmm-ovf              # Build VirtualBox OVA
make pmm-azure            # Build Azure image
make pmm-digitalocean     # Build DigitalOcean image
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
