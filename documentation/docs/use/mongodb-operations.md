# Install and monitor MongoDB clusters

!!! warning "Developer Preview"
    Operations for MongoDB is a Developer Preview feature in PMM 3.10.0. Use it for testing and feedback only, not in production.

If you are a PMM Admin user, use Operations for MongoDB to install Percona Server for MongoDB replica sets on empty nodes without SSH access and to check the health of every MongoDB cluster that PMM monitors.

## Prerequisites

Operations for MongoDB runs on [PMM Extensions](../apps/deploy-pmm-extensions.md). It isn't supported on AMI, PMM HA, or deployments with an external PostgreSQL database.

Before you enable Operations for MongoDB, prepare PMM Server:
{.power-number}

1. [Deploy PMM Extensions with Docker](../apps/deploy-pmm-extensions.md). This setup already sets `PMM_ENABLE_EXTENSIONS=1`, `PMM_ENABLE_NOMAD=1`, and `PMM_PUBLIC_ADDRESS`.
2. Publish port `4647` on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes connect to this port to receive scan and install jobs. Without it, remote nodes cannot receive jobs.
3. Connect PMM Server to PMM Extensions, so that PMM can turn on Operations for MongoDB manually: 

    1. Generate a token, for example with `openssl rand -hex 32`. Treat it as a password.
    2. Set these variables, using the same token on both containers:

        - On `pmm-server`: `PMM_EXTENSIONS_URL=http://pmm-extensions:9000` and `PMM_EXTENSIONS_TOKEN=<token>`
        - On `pmm-extensions`: `EXTENSIONS_INTERNAL_TOKEN=<token>`

        For Docker Compose, keep the token in your `.env` file as `PMM_EXTENSIONS_TOKEN`:

        ```yaml
        services:
          pmm-server:
            environment:
              PMM_EXTENSIONS_URL: http://pmm-extensions:9000
              PMM_EXTENSIONS_TOKEN: ${PMM_EXTENSIONS_TOKEN}
          pmm-extensions:
            environment:
              EXTENSIONS_INTERNAL_TOKEN: ${PMM_EXTENSIONS_TOKEN}
        ```

    3. Recreate both containers so they pick up the new variables. Restarting them isn't enough. With Docker Compose, run `docker compose up -d`.

## Enable Operations for MongoDB

You can enable Operations for MongoDB from the PMM settings or with an environment variable. Until you enable it, **MongoDB > Operations** doesn't appear in the sidebar and PMM doesn't scan nodes:

=== "PMM settings"

    Use this method to enable Operations for MongoDB on a running PMM Server without restarting it. Go to **Configuration > Settings > Advanced Settings**, turn on **Operations for MongoDB**, and click **Apply changes**.

    If PMM can't reach PMM Extensions, the setting doesn't turn on. Check your PMM Extensions deployment and try again.

=== "Environment variable"

    Use this method to keep Operations for MongoDB enabled every time PMM Server starts, for example in a Docker Compose file or a startup script.

    Start PMM Server with `PMM_ENABLE_OM=1`. You then can't change the setting in the PMM UI: **Apply changes** returns an error.

After you enable it, **MongoDB > Operations** appears in the sidebar with a **Preview** label and following pages:

- **Fleet**, where you can check health of every monitored MongoDB cluster.
- **Nodes** shows every node that PMM knows about, and whether it's ready to be scanned and installed on.
- **Automations** shows history of installs and scans.
- **Settings**, where you can check how PMM scans your nodes and what it collects.
## Scan nodes

PMM scans nodes to find out what runs on them and whether you can install MongoDB on them. A node shows no data until PMM has scanned it.

To scan nodes now, go to **MongoDB > Operations > Nodes** and do one of the following:

- To scan every node, click **Scan all**.
- To scan one node, click **Scan** in the node's row. Use this right after you add a node, so you don't wait for the next scheduled scan.

### Change the scan schedule

When you enable Operations for MongoDB, PMM turns on automatic scans and scans every node every 10 minutes. PMM turns automatic scans back on whenever it restarts.

To change how often PMM scans the nodes:
{.power-number}

1. Go to **MongoDB > Operations > Settings** and select the **General** tab.
2. In **Scan every** and **Period**, set how often PMM scans the nodes. Each scan sends a job to every node.
3. Click **Save**. The changes take effect without a restart.

### Choose what scans collect

The **Connect to MongoDB** setting on the **General** tab controls whether scans connect to `mongod`:

- **On**: scans also read each `mongod`'s replica set state and running version. This needs credentials.
- **Off**: scans collect only process and OS details, and the installed MongoDB version. This needs no credentials.

### Provide credentials for MongoDB instances

When **Connect to MongoDB** is on, which is the default, scans authenticate to `mongod`. Scans don't use the monitoring credentials that PMM holds, even for instances that PMM already monitors.

On every node that runs `mongod`, create the `/root/.mongodb_uri` file with one line:

```
mongodb://<user>:<password>@<host>:<port>
```

Scans use only the user and password from this file and authenticate against the `admin` database. The user needs the `clusterMonitor` role. The PMM monitoring user already has this role.

Nodes without MongoDB don't need this file.

### Troubleshoot scans

The summary at the top of the **Nodes** page shows how many nodes are eligible for automation, have no database, can't be scanned, or are failing. The **Automation** column shows each node's status. When a node shows **Needs attention**, the reason appears under the status.

If a node still shows no data after a scan, go to **MongoDB > Operations > Automations**, select the **Scans** tab, and check for scan errors.

## View fleet health

Check fleet health as part of your regular operational checks, or when a cluster isn't behaving as expected. It helps you spot replication lag early and confirm that your clusters run the versions you expect.

To view fleet health:
{.power-number}

1. Go to **MongoDB > Operations > Fleet**. The **Clusters** tab opens. Clusters are grouped by environment, and the top of the page shows how many environments, clusters, and services PMM monitors, and how many services are up or down.
2. For each cluster, check the following columns:

    - **Health**: the overall state of the cluster, for example **Healthy**.
    - **Services**: the number of services in the cluster.
    - **Process**: the member types in the cluster, for example routers and shards.
    - **Max repl. lag**: the highest replication lag across members.
    - **Min oplog window**: the smallest oplog window across members.

3. To see a cluster's members, expand its row.
4. To see individual services, select the **Services** tab.

The page shows when its data was last refreshed and when the nodes were last scanned. To load the latest data, click **Refresh**.

## Install a replica set

You can install a replica set with one member (1 node) or three members (3 nodes). PMM must have scanned each node at least once before you can install on it.

### Node requirements

Each node must meet the following requirements. PMM checks them during scans and reports the results, so you can fix issues before you install.

| Requirement | Needed for scans | Needed for installs |
|---|---|---|
| PMM Client 3.2.0 or later, installed from the DEB or RPM package | Yes | Yes |
| `pmm-agent` running as root (the default for the package's systemd service) | Yes | Yes |
| Writable cgroups and the `iproute` package | Yes | Yes |
| Outbound TCP access to PMM Server on port `4647` | Yes | Yes |
| `python3` with the `venv` module (`python3-venv` on Ubuntu; included on Rocky Linux) | Yes | Yes |
| Access to PyPI or a configured pip mirror (the scan installs `pymongo`) | Yes | Yes |
| HTTPS access to the `/extensions` URL on PMM Server | No | Yes |
| HTTPS access to `repo.percona.com` and OS package mirrors | Reported only | Yes |
| Ubuntu or Rocky Linux, running systemd | No | Yes |

**Reported only** means that the scan reports whether the requirement is met but doesn't fail if it isn't. The install fails if it isn't met.

PMM doesn't check the following requirements before the install. If either is missing, the install fails when PMM forms the replica set, and PMM rolls it back:

- **Resolvable node names**: replica set members address each other by their PMM node names. Every member must resolve the node names of all members, including its own, through DNS or `/etc/hosts`.
- **Network access between members**: the members can reach each other on the MongoDB port.

### Node checks

Before an install starts, PMM checks each node. If a node fails a check, it isn't eligible for automation, and the reason appears in the **Automation** column on the **Nodes** page.

Each node must meet the following conditions:

- **Supported OS**: the node runs Ubuntu or Rocky Linux, based on the `ID` value in `/etc/os-release`. RHEL, AlmaLinux, Oracle Linux, and Debian aren't supported. All nodes in one install must run the same OS.
- **No existing MongoDB**: `mongod` isn't installed or running, `/etc/mongod.conf` doesn't exist, and PMM has no MongoDB service registered for the node.
- **Data directory**: the data directory (default `/var/lib/mongo`) is missing or empty, and has at least 5 GiB free.
- **Free port**: nothing else listens on the chosen port (default `27017`).
- **Not the PMM Server host**: the node isn't the host that runs PMM Server.

### Reduce security exposure during the install

During the install, `mongod` runs without authentication. PMM turns authentication on for each node only in the final step.

Until that step, any host that can reach the MongoDB port on a node can connect without credentials. A user created in this window keeps working after authentication is turned on.

To reduce the risk:

- Make sure the nodes aren't reachable from untrusted networks before you start the install.
- Keep the default **Listen on** setting. By default, each member listens only on its own address and on `127.0.0.1`. Listen on all interfaces only if you need it.
- Don't turn off Operations for MongoDB or restart PMM Server while an install is running. If the install stops, `mongod` stays without authentication until the install resumes.

### Run the install

To install a MongoDB replica set:
{.power-number}

1. Go to **MongoDB > Operations > Nodes**. To show only nodes without a database, select the **Not monitored** filter.

2. Select one node for a single-member replica set, or three nodes for a three-member replica set, and click **Install MongoDB**. To install on a single node, you can also click **Install MongoDB** in its row.

3. Fill in the form:

    - **MongoDB version**: 7.0 (default) or 8.0, installed as Percona Server for MongoDB.
    - **Replica set name**.
    - **Data path** (default `/var/lib/mongo`), **Log path** (default `/var/log/mongodb/mongod.log`), and **Port** (default `27017`).

        !!! note "Keep the default port"
            In PMM 3.10.0, if you install on a port other than `27017`, PMM still registers the service for monitoring on port `27017`, so monitoring doesn't work. For details, see [PMM-15764](https://perconadev.atlassian.net/browse/PMM-15764).

    - **Listen on**: by default, each member listens on its own address. PMM always adds `127.0.0.1`.
    - Optional: per-member **priority**, **votes**, **hidden**, and **delay** settings for replica set elections.
    - Optional: **environment** and **cluster** labels for PMM.
    - **Administrator user**: the default name is `admin`, with the `root` role. Enter a password of at least 8 characters, or use the generated one.

4. Review the settings and start the install.

    !!! warning "Save the generated password"
        PMM shows a generated password only once and deletes its copy after it creates the user. Save the password before you close the confirmation screen.

5. To follow progress, go to **MongoDB > Operations > Automations** and expand the install on the **Installs** tab. Installs are listed newest first. PMM shows the status of each node. When all nodes finish, PMM forms the replica set, creates the users, and turns on authentication.

The install takes a few minutes per node, mostly for downloading packages. When it finishes, the replica set appears on the **Fleet** page and PMM starts monitoring it.

### If an install fails

If an install step fails, PMM retries it once. If the retry also fails, PMM rolls back what it installed on all nodes: packages, configuration, and data. PMM doesn't retry failed pre-checks.

To stop an install that's running, expand it on the **Installs** tab and abort it. PMM then rolls back the install in the same way.

## Limitations

This Developer Preview has the following limitations:

- **Replica set size**: you can install only replica sets with exactly 1 or 3 members, not sharded clusters or standalone instances. The **Fleet** page still shows sharded clusters that PMM already monitors.
- **Encryption**: no TLS and no encryption at rest. Members authenticate to each other with a keyfile. TLS and encryption at rest are planned for a future release.
- **Operating systems**: Ubuntu and Rocky Linux only, installed from Percona packages.
- **Access**: only Admin users can see **MongoDB > Operations** and start installs.