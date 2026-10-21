# MongoDB Operations

!!! warning "Developer Preview"
    Operations for MongoDB is still under development and not recommended for production environments. Expect rough edges, incomplete flows, and UI changes in future releases.

MongoDB Operations gives you a fleet view of every MongoDB cluster PMM monitors and enables you to install Percona Server for MongoDB onto empty nodes directly from PMM, without SSH access. Admin users can access it under **Operations** in the side menu, with sub-pages **Fleet**, **Nodes**, **Automations**, and **Settings**.

MongoDB Operations requires [PMM Extensions deployed with Docker](../apps/deploy-pmm-extensions.md) and does not work on AMI, PMM HA, or external PostgreSQL deployments.

## Pages

MongoDB Operations has four pages:

- **Fleet** — every monitored MongoDB cluster and the state of its members. Two tabs: **Clusters** and **Services**.
- **Nodes** — every node Operations knows about, including nodes with no database. Shows whether each node is ready to be scanned and installed on. This is where installs start.
- **Automations** — every MongoDB install Operations has run, newest first. Two tabs: **Installs** and **Scans**.
- **Settings** — how Operations scans your nodes and what it collects from them. Three tabs: **General**, **Scanning**, and **Advanced**.

## How it works

MongoDB Operations uses **scans**: a job PMM runs on each node to collect what metrics do not show, such as the OS, installed MongoDB packages, running `mongod` processes, and whether the node can reach Percona's repository. Fleet and Nodes both depend on scan data, so scans must succeed before either page shows useful results. Scans can fail if node requirements are not met; check the **Scans** tab in **Automations** for the error details.

Automatic node scans are off by default. To enable them, go to **Operations > Settings > General** and turn on **Automatic node scans**. You can also trigger a scan manually from the **Nodes** page at any time.

## Enable MongoDB Operations

Before you start, make sure that:

- PMM Extensions is deployed with Docker as described in [Deploy PMM Extensions with Docker](../apps/deploy-pmm-extensions.md). That setup already sets `PMM_ENABLE_EXTENSIONS=1`, `PMM_ENABLE_NOMAD=1`, and `PMM_PUBLIC_ADDRESS`.
- Port **4647** is published on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes connect to this port to receive scan and install jobs. Without it, remote nodes cannot receive jobs.

To enable MongoDB Operations:
{.power-number}

1. Go to **Configuration > Settings > Advanced Settings**.
2. Turn on **Operations for MongoDB**.

Alternatively, start PMM Server with `PMM_ENABLE_OM=1`. Scans run only while Operations is on.

## View your MongoDB fleet

To view your MongoDB fleet:
{.power-number}

1. Go to **Operations > Fleet** to see clusters with health status, version, replication lag, and oplog window.
2. Go to **Operations > Nodes** to see every node and whether it is ready for an install. Database health is on **Fleet**, not **Nodes**.
3. Go to **Operations > Automations > Scans** for scan history. To run a scan immediately, for example right after adding a new node, use the scan action on the **Nodes** page.

### Scan credentials for existing MongoDB

When **Connect to MongoDB** is on in **Operations > Settings > General**, the scan uses the monitoring credentials PMM already holds for each service to collect replica set state and running version. Turn it off to collect process and OS facts only — no credentials needed, and the installed version is still reported.

For services PMM does not yet monitor, the scan reads credentials from `/root/.mongodb_uri` on the node. The file must contain one line in this format:

```
mongodb://<user>:<password>@<host>:<port>
```

Only the user and password are used, authenticated against `admin`. The user needs the `clusterMonitor` role, which the PMM monitoring user already has. Nodes with no MongoDB do not need this file.

## Install a MongoDB replica set

See [Install a MongoDB replica set](install-mongodb-replica-set.md).

## Limitations

- Replica sets only, of exactly 1 or 3 members. No sharded clusters.
- No TLS and no encryption at rest. Members authenticate to each other with a keyfile.
- Ubuntu and Rocky Linux only, installed from Percona's packages.
- Admin-only access. Viewer and Editor roles can see fleet data but cannot trigger installs.
- Not supported on AMI, PMM HA, or external PostgreSQL deployments.
