# Connect Linux databases to PMM

## Supported Linux distributions

PMM Client supports collecting system metrics from various Linux distributions:

- Red Hat/CentOS/Oracle Linux 8 and 9
- Amazon Linux 2023 (native support added in PMM 3.2.0)
- Debian 11 (Bullseye) and 12 (Bookworm)
- Ubuntu 22.04 (Jammy) and 24.04 (Noble)

## Add Linux monitoring

When you register a node using the PMM Client, system metrics collection is enabled by default:

```bash
pmm-admin config --server-url=https://admin:admin@pmm-server-ip:443
```

## Viewing Linux metrics

To view collected Linux metrics:
{.power-number}

1. Go to the **Operating System (OS) > Overview** dashboard.
2. Select your node from the **Node Names** dropdown menu.
3. Explore additional OS-specific dashboards for more detailed metrics:
    - **OS > Node Summary**
    - **OS > CPU Utilization Details**
    - **OS > Disk Details**
    - **OS > Memory Details**
    - **OS > Network Details**
    - **OS > Node Temperature Details**
    - **OS > NUMA Details**
    - **OS > Processes Details**

## Related topics

- [Install PMM Client](../../install-pmm-client/index.md)
- [Operating System dashboard reference](../../../reference/dashboards/dashboard-node-summary.md)
- [Troubleshooting PMM](../../../troubleshoot/index.md)
