# Prerequisites

1. Check your system [requirements](..//..//plan-pmm-installation/hardware_and_system.md#server-requirements).

2. Configure your [network](..//..//plan-pmm-installation/network_and_firewall.md).

3. Authenticate using Service Accounts.

    While adding clients to the PMM Server, you use the `admin` user. However, if you change the password for the admin user from the PMM UI, then the clients will not be able to access PMM. Also, due to multiple unsuccessful login attempts Grafana will lock out the `admin` user. The solution is to use [Service Accounts](../../api/authentication.md) for authentication. You can use Service Accounts as a replacement for basic authentication and API keys.