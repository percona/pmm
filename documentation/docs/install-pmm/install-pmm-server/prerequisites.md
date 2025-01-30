# Prerequisites

Before setting up PMM Server, ensure you complete the following requirements:
{.power-number}

1. Check that your system meets the [hardware and software requirements](../plan-pmm-installation/hardware_and_system.md).

2. Configure your [network settings](../plan-pmm-installation/network_and_firewall.md).

3. Use Grafana [Service Accounts](../../api/authentication.md) for secure and consistent authentication. 

PMM 3 uses Grafana service accounts for authentication instead of API keys, which provide fine-grained access control and enhanced security.

Service accounts prevent issues such as Clients losing access due to admin password changes or account lockouts caused by multiple failed login attempts.