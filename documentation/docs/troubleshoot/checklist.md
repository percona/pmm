# Troubleshooting checklist

The following questions might help you identify the origin of the problem while using Percona Monitoring and Management:
{.power-number}

1. Are you using the latest PMM version?
2. Did you check the known issues section in the Release Notes for that particular PMM release?
3. Are you receiving any error messages?
4. Do the logs contain any messages about the problem? See Message logs and Trace logs for more information.
5. Does the problem occur while configuring PMM, such as:
     - Does the problem occur while you configure a specific function?
     - Does the problem occur when you perform a particular task?
6. Are you using the recommended [authentication](../api/authentication.md#authenticate) method?
7. Does your system’s firewall allow TCP traffic on the [ports](../install-pmm/plan-pmm-installation/network_and_firewall.md#essential-ports) used by PMM?
8. Have you allocated enough [disk space](https://www.percona.com/blog/2017/05/04/how-much-disk-space-should-i-allocate-for-percona-monitoring-and-management/) for installing PMM? If not, check the disk allocation space.
9. Are you using a Technical Preview feature? Technical Preview features are not production-ready and should only be used in testing environments. For more information, see the relevant Release Notes.
10. For installing the PMM client, are you using a package other than a binary package without root permissions?
11. Is your [PMM Server](../install-pmm/install-pmm-server/index.md) installed and running with a known IP address accessible from the client node?
12. Is the [PMM Client](../install-pmm/install-pmm-client/index.md) installed, and is the node [registered with PMM Server](../install-pmm/register-client-node/index.md)?
13. Is PMM Client configured correctly and has access to the config file?
14. For monitoring MongoDB, do you have adminUserAnyDatabase or superuser role privilege to any database servers you want to monitor?
15. For monitoring Amazon RDS using PMM, is there too much latency between PMM Server and the Amazon RDS instance?
16. Have you upgraded the PMM Server before you upgraded the PMM Client? If yes, there might be configuration issues, thus leading to failure in the client-server communication, as PMM Server might not be able to identify all the parameters in the configuration.
17. Is the PMM Server version higher than or equal to the PMM Client version? Otherwise, there might be configuration issues, thus leading to failure in the client-server communication, as PMM Server might not be able to identify all the parameters in the configuration.


