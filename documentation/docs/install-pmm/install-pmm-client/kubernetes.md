# Run PMM Client as a pod in a Kubernetes Deployment

The [PMM Client Docker image](https://hub.docker.com/r/percona/pmm-client/tags/) can be deployed as a pod in Kubernetes, provides a convenient way to run PMM Client as a pre-configured container without installing software directly on your host system.

Using the Kubernetes pod approach offers several advantages:

- no need to install PMM Client directly on your host system
- consistent environment across different operating systems
- simplified setup and configuration process
- automatic architecture detection (x86_64/ARM64)
- [centralized configuration management](../install-pmm-server/deployment-options/docker/env_var.md#configure-vmagent-variables) through PMM Server environment variables

## Prerequisites

Complete these essential steps before installation:
{.power-number}

1. Install [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/).

2. Check [system requirements](prerequisites.md) to ensure your environment meets the minimum criteria.

3. [Install and configure PMM Server](../install-pmm-server/index.md) as you'll need its IP address or hostname to configure the Client.

4. [Set up firewall rules](../plan-pmm-installation/network_and_firewall.md) to allow communication between PMM Client and PMM Server.

5. [Create database monitoring users](prerequisites.md#database-monitoring-requirements) with appropriate permissions for the databases you plan to monitor.

## Installation and setup

### Deploy PMM Client

Follow these steps to deploy PMM Client using `kubectl`:
{.power-number}

1. Create a Persistent Volume to store PMM Client data between pod restarts:

2. Create a Secret to store the credentials for PMM Server authentication

3. Deploy PMM Client pod and configure the [pmm-agent](../../use/commands/pmm-agent.md) in Setup mode to connect to PMM Server

## Verify the connection

## View your monitored node

To confirm your node is being monitored:
{.power-number}

  1. Go to the main menu and select **Operating System (OS) > Overview**.

  2. In the **Node Names** drop-down menu, select the node you recently registered.

  3. Modify the time range to view the relevant data for your selected node.

!!! danger alert alert-danger "Danger"
    `pmm-agent.yaml` contains sensitive credentials and should not be shared.
