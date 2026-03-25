# Deploy PMM Server as a Virtual Appliance (OVA)

!!! warning "Deprecation notice"
    OVF virtual appliance images are deprecated starting with PMM 3.7.0. OVF images will be formally deprecated in PMM 3.8 (expected May 2026) and will no longer be produced after PMM 3.9 (expected July 2026).

    If you currently run PMM on a virtual appliance, plan your migration to a supported deployment method:

    - [Install PMM Server with Docker](../docker/index.md)|(recommended) 
    - [Install PMM Server with Podman](../podman/index.md)
    - [Install PMM Server with Helm on Kubernetes](../helm/index.md)
    
    Existing OVF deployments will continue to work. You can still update PMM within your current virtual appliance using CLI upgrade methods, but no new OVF images will be released after PMM 3.9.0.

Deploy PMM Server as a pre-configured virtual machine when you need a standalone monitoring solution with minimal setup. The virtual appliance is ideal for environments where container solutions aren't preferred or for evaluation purposes.

## When to choose OVA deployment
Choose OVA deployment you: 

- prefer traditional VM-based deployments over containers
- need a solution that works with existing virtualization infrastructure
- want minimal configuration steps for quick evaluation
- environment has limited or no internet connectivity

## Terminology

When working with the PMM Server virtual appliance, it's helpful to understand these terms:

- **Host**: The desktop or server machine running the hypervisor
- **Hypervisor**: Software (e.g., VirtualBox) that runs the guest OS as a virtual machine
- **Guest VM**: Virtual machine running PMM Server (Oracle Linux 9.3)

## OVA file details

| Item | Value |
|------|-------|
| Download page | https://www.percona.com/downloads/pmm/{{release}}/ova |
| File name | `pmm-server-{{release}}.ova` |
| VM name | `pmm-Server-{{release_date}}-N` (`N`=build number) |

## VM specifications

The PMM Server virtual appliance comes pre-configured with these specifications:

| Component | Value |
|-----------|-------|
| OS | Oracle Linux 9.3 |
| CPU | 1 |
| Base memory | 4096 MB |
| Disks | LVM, 2 physical volumes |
| Disk 1 (`sda`) | VMDK (SCSI, 40 GB) |
| Disk 2 (`sdb`) | VMDK (SCSI, 400 GB) |

!!! note
    You can adjust CPU and memory resources after deployment to match your monitoring needs.

## System requirements

For optimal performance, we recommend:

=== "Minimum (1-30 nodes)"
    - **CPU**: 4 cores
    - **Memory**: 8 GB
    - **Disk**: 100 GB

=== "Recommended (31-100 nodes)"
    - **CPU**: 8 cores
    - **Memory**: 16 GB
    - **Disk**: 200 GB

=== "Large (100+ nodes)"
    - **CPU**: 16+ cores
    - **Memory**: 32+ GB
    - **Disk**: 500+ GB

## Hypervisor compatibility

The PMM Server OVA is compatible with VirtualBox 6.0 and later. 

## Network requirements

Ensure your network environment allows:

- Outbound internet access for updates (optional)
- Access to monitored database instances
- Access from client browsers to the PMM Server web interface
- Standard ports: 443 (HTTPS), 80 (HTTP, redirects to HTTPS)

See [Network and firewall requirements](../../../plan-pmm-installation/network_and_firewall.md) for full details.

## Default users

PMM Server comes with two pre-configured user accounts that you must secure immediately after installation:

- **admin** (default password: `admin`)
- **root** (default password: `percona`)

Change these default passwords to strong, unique passwords during your first login to prevent unauthorized access.

## Next steps

After reviewing the requirements:

- [Download the PMM Server OVA file](download_ova.md)
- [Deploy on VirtualBox](virtualbox.md)
- [Configure environment variables](env_var.md) to customize PMM Server behavior