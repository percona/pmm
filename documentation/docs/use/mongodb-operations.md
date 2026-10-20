# MongoDB Operations

!!! warning "Developer Preview"
    MongoDB Operations is a Developer Preview in PMM 3.10.0, intended for evaluation by Percona engineers and selected customers. It is not recommended for production use. Expect rough edges, incomplete flows, and UI changes in future releases.

MongoDB Operations gives you a fleet view of every MongoDB cluster PMM monitors and lets you install Percona Server for MongoDB onto empty nodes directly from PMM, without SSH access. It is visible to PMM Admin users only, under **MongoDB > Operations** in the sidebar.

MongoDB Operations requires [PMM Extensions deployed with Docker](../apps/deploy-pmm-extensions.md). It does not work on AMI, PMM HA, or external PostgreSQL deployments.

## Pages

MongoDB Operations has four pages:

- **Fleet** — your MongoDB clusters and services, with health status, versions, replication lag, and oplog window. Two tabs: **Clusters** and **Services**.
- **Nodes** — every node running PMM Client, including nodes with no database yet. This is where installs start.
- **Automations** — history of installs and scans. Two tabs: **Installs** and **Scans**.
- **Settings** — scan frequency, timeouts, and concurrency.

## How it works

MongoDB Operations uses **scans**: a job PMM runs on each node every 10 minutes to collect what metrics do not show, such as the OS, installed MongoDB packages, running `mongod` processes, and whether the node can reach Percona's repository. Fleet and Nodes both depend on scan data, so scans must succeed before either page shows useful results. Scans can fail if node requirements are not met; check the **Automations > Scans** tab for the error details.

## Enable MongoDB Operations

Before you start, make sure that:

- PMM Extensions is deployed with Docker as described in [Deploy PMM Extensions with Docker](../apps/deploy-pmm-extensions.md). That setup already sets `PMM_ENABLE_EXTENSIONS=1`, `PMM_ENABLE_NOMAD=1`, and `PMM_PUBLIC_ADDRESS`.
- Port **4647** is published on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes connect to this port to receive scan and install jobs. Without it, remote nodes cannot receive jobs.
- The PMM Extensions base URL is set to `https://<PMM_PUBLIC_ADDRESS>/extensions` using `BASE_URL` (with `docker run`) or `PMM_EXTENSIONS_BASE_URL` (with Compose). Use an address that PMM Client nodes can reach. This is required for installs.

To enable MongoDB Operations:
{.power-number}

1. Go to **Configuration > Settings > Advanced Settings**.
2. Turn on **Operations for MongoDB**.

Alternatively, start PMM Server with `PMM_ENABLE_OM=1`. Scans run only while Operations is on.

## View your MongoDB fleet

To view your MongoDB fleet:
{.power-number}

1. Go to **MongoDB > Operations > Fleet** to see clusters with health status, version, replication lag, and oplog window.
2. Go to **Nodes** to see every node and whether it is ready for an install.
3. Go to **Automations > Scans** for scan history. To run a scan immediately, for example right after adding a new node, use the scan action on the **Nodes** page.

### Scan credentials for existing MongoDB

To collect database details from a `mongod` that requires authentication, the scan reads credentials from `/root/.mongodb_uri` on the node. The file must contain one line in this format:

```
mongodb://<user>:<password>@<host>:<port>
```

Only the user and password are used, authenticated against `admin`. The user needs the `clusterMonitor` role, which the PMM monitoring user already has. Without this file, the scan still reports the node and its processes, but database details show an error. Nodes with no MongoDB do not need this file.

## Install a MongoDB replica set

See [Install a MongoDB replica set](install-mongodb-replica-set.md).

## Limitations

- Replica sets only, of exactly 1 or 3 members. No sharded clusters.
- No TLS and no encryption at rest. Members authenticate to each other with a keyfile.
- Ubuntu and Rocky Linux only, installed from Percona's packages.
- Admin-only access. Viewer and Editor roles can see fleet data but cannot trigger installs.
- Not supported on AMI, PMM HA, or external PostgreSQL deployments.
