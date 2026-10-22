# Install and monitor MongoDB clusters

!!! warning "Developer Preview"
    Operations for MongoDB is a Developer Preview feature in PMM 3.10.0. Use it for testing and feedback only, not in production.

As a PMM Admin, you can use Operations for MongoDB to check the health of every MongoDB cluster that PMM monitors, and to install Percona Server for MongoDB replica sets on empty nodes without SSH access. Before you start, review the [limitations](#limitations) that apply in this Developer Preview release.

## Prepare PMM Server

Operations for MongoDB runs on [PMM Extensions](../apps/deploy-pmm-extensions.md). It isn't supported on AMI, PMM HA, or deployments with an external PostgreSQL database.

Before you enable Operations for MongoDB, prepare PMM Server:
{.power-number}

1. [Deploy PMM Extensions with Docker](../apps/deploy-pmm-extensions.md). This setup already sets `PMM_ENABLE_EXTENSIONS=1`, `PMM_ENABLE_NOMAD=1`, and `PMM_PUBLIC_ADDRESS`.
2. Publish port `4647` on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes connect to this port to receive scan and install jobs. Without it, remote nodes can't receive jobs.
3. Connect PMM Server to PMM Extensions. This step is required in PMM 3.10.0 only (see [PMM-15772](https://perconadev.atlassian.net/browse/PMM-15772)). Without it, you can't turn on Operations for MongoDB:

    1. Generate a token, for example with `openssl rand -hex 32`. Treat it as a password.
    2. Set these variables, using the same token on both containers:

        - On `pmm-server`: `PMM_EXTENSIONS_URL=http://pmm-extensions:9000` and `PMM_EXTENSIONS_TOKEN=<token>`
        - On `pmm-extensions`: `EXTENSIONS_INTERNAL_TOKEN=<token>`

        If you use Docker Compose, store the token in your `.env` file as `PMM_EXTENSIONS_TOKEN` and reference it from both services:

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

    3. Recreate both containers so they pick up the new environment variables. A simple restart won't work, because Docker only reads environment variables when it creates a container. If you use Docker Compose, run `docker compose up -d`, which recreates any container whose configuration has changed.

### Enable Operations for MongoDB

Until you enable Operations for MongoDB, **MongoDB > Operations** doesn't appear in the side menu and PMM doesn't scan your nodes. You can enable it via PMM settings or an environment variable:

=== "via PMM settings"

    Use this method to enable Operations for MongoDB on a running PMM Server, without a restart:
    {.power-number}

    1. Go to **Configuration > Settings > Advanced Settings**.
    2. Turn on **Operations for MongoDB**.
    3. Click **Apply changes**.

    If PMM can't reach PMM Extensions, the setting stays off. Check your PMM Extensions deployment and try again.

=== "via environment variable"

    Use this method to keep Operations for MongoDB on every time PMM Server starts, for example from a Docker Compose file or a startup script.

    Start PMM Server with `PMM_ENABLE_OM=1`. While this variable is set, you can't change the setting in the UI. The **Operations for MongoDB** toggle stays visible, but **Apply changes** returns an error.

After you enable it, **MongoDB > Operations** appears in the side menu with a **Preview** label. It includes these pages:

- **Fleet**: check the health of every MongoDB cluster that PMM monitors.
- **Nodes**: see every node PMM knows about and whether it's ready to be scanned or have Percona Server for MongoDB installed.
- **Automations**: review the history of installs and scans.
- **Settings**: control how PMM scans your nodes and what it collects.

## Prepare your nodes

Before you scan or install on a node, make sure it meets the requirements and, if you want scans to connect to MongoDB, that you've provided credentials.

### Node requirements

PMM checks these requirements during scans and reports the results so you can fix issues before you install. **Reported only** means the scan checks and reports the requirement but doesn't block on it. The install enforces it and fails if it isn't met.

??? info "Requirements table"

    | Requirement | Scans | Installs |
    |---|---|---|
    | PMM Client 3.2.0 or later, installed from the DEB or RPM package | Required | Required |
    | `pmm-agent` running as root, which is the default for the package's systemd service | Required | Required |
    | Writable cgroups | Required | Required |
    | The `iproute` package | Required | Required |
    | Outbound TCP access to PMM Server on port `4647` | Required | Required |
    | `python3` with the `venv` module: `python3-venv` on Ubuntu, included on Rocky Linux | Required | Required |
    | Access to PyPI or a configured pip mirror, because scans install `pymongo` | Required | Required |
    | HTTPS access to `repo.percona.com` and OS package mirrors | Reported only | Required |
    | HTTPS access to the `/extensions` URL on PMM Server | Not needed | Required |

### Provide MongoDB credentials for scans

Scans log in to MongoDB on each node to read the replica set state and the running MongoDB version. They don't use the monitoring credentials that PMM already holds, so you need to provide credentials even for instances that PMM monitors.

To provide credentials:
{.power-number}

1. Choose a MongoDB user with the `clusterMonitor` role. You can reuse the PMM monitoring user, which already has this role.
2. On every node that runs MongoDB, create the `/root/.mongodb_uri` file with one line:

    ```
    mongodb://<user>:<password>@<host>:<port>
    ```

    Scans read only the user and password from this line and log in to the `admin` database.

3. To check the credentials, go to **MongoDB > Operations > Nodes** and click **Scan** in the node's row.

Nodes without MongoDB don't need this file.

#### Scan without credentials

To skip MongoDB login entirely, turn off **Connect to MongoDB** in **MongoDB > Operations > Settings > General**. With it off, scans collect only process and OS details and the installed MongoDB version, without connecting to `mongod`. No credentials needed, but scans won't report replica set state or running version.

## Scan nodes

A scan checks what runs on a node and whether you can install MongoDB on it. Until PMM scans a node, the node shows no data.

PMM scans every node automatically, every 10 minutes by default.

### Run a scan now

To scan right away, for example after you add a node, go to **MongoDB > Operations > Nodes** and do one of the following:

- To scan every node, click **Scan all**.
- To scan one node, click **Scan** in its row.

### Change the scan schedule

By default, PMM scans every node every 10 minutes. Each scan sends a job to every node. To scan more or less often:
{.power-number}

1. Go to **MongoDB > Operations > Settings** and select the **General** tab.
2. In **Scan every** and **Period**, set the time between scans.
3. Click **Save**. The change takes effect without a restart.

!!! note "Keep automatic scans on"
    In PMM 3.10.0, turning off **Automatic node scans** also blocks **Scan** and **Scan all** on the **Nodes** page. PMM turns automatic scans back on the next time it restarts.

## Install a replica set

PMM supports single-member and three-member replica sets. Each node must have at least one completed scan before you can select it for installation.

### Before you install

Check your nodes and reduce the security exposure before you start the install.

#### Check nodes

Make sure each node meets the following conditions before you start. If a node doesn't, PMM blocks it and shows the reason in the **Automation** column on the **Nodes** page:

- **Supported OS**: Ubuntu or Rocky Linux only, matched on `ID` in `/etc/os-release`. RHEL, AlmaLinux, Oracle Linux, and Debian aren't supported. All nodes in one install must run the same OS.
- **No existing MongoDB**: `mongod` must not be installed or running, `/etc/mongod.conf` must not exist, and PMM must have no MongoDB service registered for the node.
- **Free data directory**: the data directory (default `/var/lib/mongo`) must be missing or empty, with at least 5 GiB free.
- **Free port**: nothing else must be listening on the chosen port (default `27017`).
- **Not the PMM Server host**: don't install on the host that runs PMM Server.

PMM doesn't check the following before the install starts. If they aren't met, the install fails when PMM forms the replica set and then rolls it back:

- **Resolvable node names**: replica set members address each other by their PMM node names. Every member must resolve the node names of all members, including its own, through DNS or `/etc/hosts`.
- **Network access between members**: each member can reach the other members on the MongoDB port.

#### Reduce security exposure

During the install, `mongod` runs without authentication until the final step, when PMM turns authentication on for each node. Until then, any host that can reach the MongoDB port on a node can connect without credentials, and any user it creates keeps working after authentication is on.

To reduce this risk:

- Make sure the nodes aren't reachable from untrusted networks before you start the install.
- Keep the default **Listen on** setting. By default, each member listens only on its own address and on `127.0.0.1`. Listen on all interfaces only if you need it.
- Don't turn off Operations for MongoDB or restart PMM Server while an install is running. If the install stops, `mongod` stays without authentication until the install resumes.

### Run the install

Once your nodes pass the checks and you've reduced the security exposure, you're ready to install. To install a MongoDB replica set:
{.power-number}

1. Go to **MongoDB > Operations > Nodes**.

    !!! hint ""
        To show only nodes without a database, select the **Not monitored** filter.

2. Select one node for a single-member replica set, or three nodes for a three-member replica set, and click **Install MongoDB**. To install on a single node, you can also click **Install MongoDB** in its row.

3. Fill in the form:

    - **MongoDB version**: 7.0 (default) or 8.0, installed as Percona Server for MongoDB.
    - **Replica set name**.
    - **Data path** (default `/var/lib/mongo`), **Log path** (default `/var/log/mongodb/mongod.log`), and **Port** (default `27017`).

        !!! note "Keep the default port"
            In PMM 3.10.0, PMM registers the service for monitoring on port `27017` even if you install on another port, so monitoring doesn't work. This is a [known issue](https://perconadev.atlassian.net/browse/PMM-15764).

    - **Listen on**: by default, each member listens on its own address. PMM always adds `127.0.0.1`.
    - Optional: per-member **priority**, **votes**, **hidden**, and **delay** settings for replica set elections.
    - Optional: **environment** and **cluster** labels for PMM.
    - **Administrator user**: the default name is `admin`, with the `root` role. Enter a password of at least 8 characters, or use the generated one.

4. Review the settings and start the install.

    !!! warning "Save the generated password"
        PMM shows a generated password only once and deletes its copy after it creates the user. Save the password before you close the confirmation screen.

5. Track progress in **MongoDB > Operations > Automations**. Expand the install on the **Installs** tab (installs are listed newest first) to see the status of each node. When all nodes finish, PMM forms the replica set, creates the users, and turns on authentication.

The install takes a few minutes per node, mostly for downloading packages. When it finishes, the replica set appears on the **Fleet** page and PMM starts monitoring it.

## View fleet health

Check fleet health as part of your regular operational checks, or when a cluster isn't behaving as expected. It helps you spot replication lag early and confirm that your clusters run the versions you expect.

To view fleet health:
{.power-number}

1. Go to **MongoDB > Operations > Fleet**.
2. On the **Clusters** tab, check the summary at the top: how many environments, clusters, and services PMM monitors, and how many services are up or down.
3. Clusters are grouped by environment. For each cluster, check the following columns:

    - **Health**: the overall state of the cluster, for example **Healthy**.
    - **Services**: the number of services in the cluster.
    - **Process**: the member types in the cluster, for example routers and shards.
    - **Max repl. lag**: the highest replication lag across members.
    - **Min oplog window**: the smallest oplog window across members.

4. Expand a cluster's row to find the member that causes poor health or high replication lag.
5. Select the **Services** tab to check whether a specific service is up or down.

The page shows when its data was last refreshed and when the nodes were last scanned. To load the latest data, click **Refresh**.

## Troubleshoot

### Scans

If a node shows no data or **Needs attention** in the **Automation** column, check the reason that appears under the status and fix it on the node.

If the reason isn't clear, go to **MongoDB > Operations > Automations**, select the **Scans** tab, and check the scan errors for that node.

### Installs

If an install step fails, PMM retries it once. If the retry also fails, PMM rolls back what it installed on all nodes: packages, configuration, and data. PMM doesn't retry failed pre-checks.

To stop an install that's running, expand it on the **Installs** tab and abort it. PMM then rolls back the install in the same way.

## Limitations

As a Developer Preview in PMM 3.10.0, Operations for MongoDB has the following limitations:

- You can install only replica sets with 1 or 3 members, not sharded clusters or standalone instances, though Fleet still shows any sharded clusters that PMM already monitors.
- No TLS or encryption at rest, so members authenticate with a keyfile until both are added in a future release.
- Installs on Ubuntu and Rocky Linux only, using Percona packages.
- Only PMM Admin users can access **MongoDB > Operations** and start installs.
