
# Integrate PMM with Percona Platform

Percona Platform brings together database distributions, support expertise, services, management, and automated insights.

Connect your PMM Servers to Percona Platform to boost the monitoring capabilities of your PMM installations and manage database deployments easier. In addition, you get access to PMM updates, automated insights, advanced advisor checks and more alert rule templates.

### Connect PMM to Percona Platform

You can connect to Percona Platform with a Percona Account or via Google or GitHub authentication. If [Percona Support](https://www.percona.com/about-percona/contact) has enabled a custom identity provider for your account, you can also log in using your company's credentials.

We recommend that you connect with a Percona Account, as this gives you access to other Percona services, including Percona Platform, Percona Customer Portal, and Community Forum. If you don’t have a Percona Account, you can create one on the [Percona Platform homepage](https://portal.percona.com/login) using the **Don't have an account? Create one?** link.

#### Prerequisites

To ensure that PMM can establish a connection to Percona Platform:

### Check that you are a member of an existing Platform organization

To check whether you are a member of an existing Platform organization:
{.power-number}

1. Log in to [Percona Platform](https://portal.percona.com) using your Percona Account. If you are connecting via GitHub, make sure you set your email address as **public** in your GitHub account. If your email address is private instead, Percona Platform cannot access it to authenticate you.

2. On the **Getting Started** page, check that the **Create organization** step shows an option to view your organization.

Contact your account administrator or create a new organization for your Percona Account if this is the case.

### Set the public address of your PMM Server
PMM automatically detects and populates the public address of the PMM Server when this is not set up. 
If you need to set it differently, go to **Settings > Advanced Settings** and edit the 
**Public Address** field.
