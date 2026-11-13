# Install and monitor MongoDB clusters

!!! warning "Developer Preview"
    Operations for MongoDB is a Developer Preview feature in PMM 3.10.0. Use it for testing and feedback only, not in production.

Operations for MongoDB enables you to check the health of every MongoDB cluster that PMM monitors, and to install Percona Server for MongoDB replica sets on empty nodes without SSH access. Before you start, review the [limitations](#limitations) that apply in this Developer Preview release.

Every signed-in PMM user can open **MongoDB > Operations** and see all of its pages, except the install form. Only PMM Admins can scan nodes, start or abort installs, remove node entries, and change settings. For other users, these controls are disabled or hidden. Any user can click **Refresh** on **Fleet**.

## Prepare PMM Server

Operations for MongoDB runs on [PMM Extensions](../apps/deploy-pmm-extensions.md). It isn't supported on AMI, PMM HA, or deployments with an external PostgreSQL database.

Before you enable Operations for MongoDB, prepare PMM Server:
{.power-number}

1. [Deploy PMM Extensions with Docker](../apps/deploy-pmm-extensions.md). This setup already sets `PMM_ENABLE_EXTENSIONS=1`, `PMM_ENABLE_NOMAD=1`, and `PMM_PUBLIC_ADDRESS`. PMM Server connects to PMM Extensions automatically.
2. Publish port `4647` on the `pmm-server` container (`-p 4647:4647`). PMM Client nodes connect to this port to receive scan and install jobs. Without it, remote nodes can't receive jobs.

### Enable Operations for MongoDB

Until you enable Operations for MongoDB, **MongoDB > Operations** doesn't appear in the side menu and PMM doesn't scan your nodes. You can enable it via PMM settings or an environment variable:

=== "via PMM settings"

    Use this method to enable Operations for MongoDB on a running PMM Server, without a restart:
    {.power-number}

    1. Go to **Configuration > Settings > Advanced Settings**.
    2. Turn on **Operations for MongoDB**.
    3. Click **Apply changes**.

    If PMM can't use PMM Extensions, the setting stays off and the error message says why, for example that PMM Extensions is unreachable. Fix the cause and try again.

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

??? info "What each node needs for scans and installs"

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

Scans log in to MongoDB on each node to read the replica set state. They don't use the monitoring credentials that PMM already holds, so you need to provide credentials even for instances that PMM monitors.

To provide credentials:
{.power-number}

1. Choose a MongoDB user with the `clusterMonitor` role. You can reuse the PMM monitoring user, which already has this role.
2. On every node that runs MongoDB, create the `/root/.mongodb_uri` file with one line:

    ```
    mongodb://<user>:<password>@<host>:<port>
    ```

    Scans read only the user and password from this line and log in to the `admin` database. Nodes without MongoDB don't need this file.

3. Make the file readable only by root, because it contains a password:

    ```sh
    chown root:root /root/.mongodb_uri
    chmod 600 /root/.mongodb_uri
    ```

4. To check the credentials, click **Scan** in the node's row on the **Nodes** page. Then go to **MongoDB > Operations > Automations**, select the **Scans** tab, and expand the latest scan. If the password is wrong, the scan shows as **Partial**, and the service appears in red with **(no answer)**. Hover over it to see the error. If the file is missing, no error is shown. Scans still report the running version, but not the replica set state.

#### Scan without credentials

To skip MongoDB login entirely, go to **MongoDB > Operations > Settings > General**, turn off **Connect to MongoDB**, and click **Save**. The change applies to the next scan, without a restart. A scan that starts within about 30 seconds of saving can still use the previous setting.

With **Connect to MongoDB** off, scans collect only process and OS details and the installed MongoDB version, without connecting to `mongod`. No credentials needed, but scans won't report replica set state or running version.

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

## Install a replica set

PMM supports single-member and three-member replica sets. Each node must have at least one completed scan before you can select it for installation.

### Before you install

Check your nodes and reduce the security exposure before you start the install.

#### Check nodes

Make sure each node meets the following conditions before you start. If a node doesn't, PMM blocks it and shows the reason in the **Automation** column on the **Nodes** page:

- **Supported OS release**: a release of Ubuntu or Rocky Linux listed in the table that follows, matched on `ID` and `VERSION_ID` in `/etc/os-release`. RHEL, AlmaLinux, Oracle Linux, and Debian aren't supported. All nodes in one install must run the same OS.
- **No existing MongoDB**: `mongod` must not be installed or running, and PMM must have no MongoDB service registered for the node.
- **Not the PMM Server host**: don't install on the host that runs PMM Server.
- **Working scans**: the node's scans must succeed. While they fail, PMM blocks the node and shows the scan error and how long scans have been failing.
- **Repository access**: the node's last scan must have reached Percona's repository at `repo.percona.com`, directly or through a proxy.
- **Package manager**: the node's last scan must have found a package manager, `apt-get` on Ubuntu or `dnf` on Rocky Linux.

PMM installs each MongoDB version only onto the OS releases that Percona's [release lifecycle for MongoDB](https://www.percona.com/release-lifecycle-percona-software-for-mongodb/) supports. Ubuntu matches on the exact release, and Rocky Linux on the major version, so Rocky Linux 9.4 counts as 9. The following table lists the supported releases:

| OS release | MongoDB 7.0 | MongoDB 8.0 |
|------------|-------------|-------------|
| Ubuntu 22.04, 24.04 | Supported | Supported |
| Rocky Linux 8, 9 | Supported | Supported |
| Rocky Linux 10 | Not supported | Supported |

PMM blocks a node whose release isn't listed, such as Ubuntu 20.04. The install form offers only the versions that every selected node supports.

PMM checks the following conditions only when the install starts, and the **Nodes** page doesn't show them. Check them yourself before you start. If a node doesn't meet them, the install fails at the pre-check, before anything on the nodes changes:

- **No MongoDB configuration**: `/etc/mongod.conf` must not exist.
- **Free data directory**: the data directory (default `/var/lib/mongo`) must be missing or empty, with at least 5 GiB free. For the default data directory, the install form already blocks a node whose last scan found less than 5 GiB free.
- **Free port**: nothing else must be listening on the chosen port (default `27017`).
- **Resolvable node names**: replica set members address each other by their PMM node names. Every member must resolve the node names of all members, including its own, through DNS or `/etc/hosts`.
- **`curl` on Ubuntu**: PMM downloads the Percona repository package with `curl`.

PMM doesn't check that each member can reach the other members on the MongoDB port, because it can only test that once `mongod` is running. If a member can't reach the others, the install fails when PMM forms the replica set, and PMM rolls it back.

#### Reduce security exposure

During the install, `mongod` runs without authentication until the final step, when PMM turns authentication on for each node. Until then, any host that can reach the MongoDB port on a node can connect without credentials, and any user it creates keeps working after authentication is on.

To reduce this risk:

- Make sure the nodes aren't reachable from untrusted networks before you start the install.
- Keep the default **Listen on** setting. By default, each member listens only on its own address and on `127.0.0.1`. Listen on all interfaces only if you need it.
- Don't turn off Operations for MongoDB or restart PMM Server while an install is running. If the install stops, `mongod` stays without authentication until the install resumes.

### Run the install

Once your nodes pass the checks and you've reduced the security exposure, you're ready to install. To install a MongoDB replica set:
{.power-number}

1. Go to **MongoDB > Operations > Nodes**. To show only the nodes that you can install onto, select the **Ready to install** filter. The **Install MongoDB** button on the **Automations** page opens the same view.

2. Select one node for a single-member replica set, or three nodes for a three-member replica set, and click **Install MongoDB**. To install on a single node, you can also click **Install MongoDB** in its row.

3. Fill in the form:

    - **MongoDB version**: 7.0 (default) or 8.0, installed as Percona Server for MongoDB. If a selected node runs Rocky Linux 10, the form offers only 8.0.
    - **Replica set name**: the name must not be in use in the same environment and cluster, by a replica set that PMM monitors or by an install that hasn't finished registering. If you leave **Cluster** empty, PMM uses the replica set name as the cluster.
    - **Data path** (default `/var/lib/mongo`), **Log path** (default `/var/log/mongodb/mongod.log`), and **Port** (default `27017`).

        !!! note "Keep the default port"
            In PMM 3.10.0, PMM registers the service for monitoring on port `27017` even if you install on another port, so monitoring doesn't work. This is a [known issue](https://perconadev.atlassian.net/browse/PMM-15764).

    - **Listen on**: by default, each member listens on its own address. PMM always adds `127.0.0.1`.
    - Optional, for three-node installs only: per-member **priority**, **votes**, **hidden**, and **delay** settings for replica set elections.
    - Optional: **environment** and **cluster** labels for PMM.
    - **Administrator user**: the default name is `admin`, with the `root` role. Enter a password of at least 8 characters, or use the generated one.

4. Review the settings and start the install.

    !!! warning "Save the generated password"
        PMM shows a generated password only once and deletes its copy after it creates the user. Save the password before you close the confirmation screen.

5. Track progress in **MongoDB > Operations > Automations**. On the **Installs** tab, expand your install to see the status of each node. The newest installs are at the top. When all nodes finish, PMM forms the replica set, creates the users, and turns on authentication.

PMM installs on all nodes in parallel, so the install takes a few minutes in total, mostly for downloading packages. When it finishes, the replica set appears on the **Fleet** page and PMM starts monitoring it.

## View fleet health

Check fleet health as part of your regular operational checks, or when a cluster isn't behaving as expected. It helps you spot replication lag early and confirm that your clusters run the versions you expect.

To view fleet health:
{.power-number}

1. Go to **MongoDB > Operations > Fleet**.
2. On the **Clusters** tab, check the summary at the top: how many environments, clusters, and services PMM monitors, and how many services are up or down.
3. Clusters are grouped by environment. For each cluster, check the following columns:

    - **Health**: the overall state of the cluster, for example **Healthy**.
    - **Services**: the number of services in the cluster.
    - **Process**: the member types in the cluster, for example routers, shards and arbiters.
    - **Max repl. lag**: the highest replication lag across members.
    - **Min oplog window**: the smallest oplog window across members.

4. Expand a cluster's row to find the member that causes poor health or high replication lag.
5. Select the **Services** tab to check whether a specific service is up or down.

The page shows when its data was last refreshed and when the nodes were last scanned. To load the latest data, click **Refresh**.

## Troubleshoot

### Pages don't load

If **Nodes**, **Automations** or **Settings** can't load and the error says **PMM Extensions did not accept the PMM session**, check the `pmm-extensions` container. PMM signs you in to PMM Extensions with your PMM session. PMM Extensions accepts that session only while `EXTENSIONS__AMBIENT_SESSION_SSO_ENABLED` is on, which is the default. If the container sets it to `false`, remove the variable and recreate the container.

### Scans

If a node shows no data or **Needs attention** in the **Automation** column, check the reason that appears under the status and fix it on the node.

If the reason isn't clear, go to **MongoDB > Operations > Automations**, select the **Scans** tab, and check the scan errors for that node.

The **Nodes** page doesn't flag rejected MongoDB credentials. To find them, look for **Partial** scans on the **Scans** tab, or check the **Services** tab as described below.

To find out why a service is failing, go to **MongoDB > Operations > Fleet** and select the **Services** tab. The **Collected** column shows the reason in red, for example **Database unreachable** or **Database authentication failed**. Click the service's status to open its details, which show the full error and what to do about it. When PMM knows which scan failed, the details also link to it. A **Down** service marked **not running** had no `mongod` or `mongos` running at its last scan.

### Installs

If an install step fails, PMM retries it once. If the retry also fails, PMM rolls back what it installed on all nodes: packages, configuration, and data. PMM doesn't retry failed pre-checks.

To stop an install that's running, expand it on the **Installs** tab and abort it. PMM then rolls back the install in the same way.

If every node on the **Nodes** page shows **PMM could not read from PMM Extensions which OS releases Operations installs onto**, PMM can't read the list of supported OS releases, so it blocks installs onto every node. The rest of the reason says why. Make sure that PMM Extensions is running and on the [same version as PMM Server](../apps/deploy-pmm-extensions.md#check-versions-before-deploying), then reload the **Nodes** page.

## Limitations

Operations for MongoDB is a Developer Preview feature in PMM 3.10.0: you can try it out, but it isn't complete or supported for production yet. It has the following limitations:

- You can install only replica sets with 1 or 3 members, not sharded clusters or standalone instances, though Fleet still shows any sharded clusters that PMM already monitors.
- No TLS or encryption at rest. Members authenticate to each other with a keyfile. TLS and encryption at rest are planned for a future release.
- Installs on the [supported releases](#check-nodes) of Ubuntu and Rocky Linux only, using Percona packages.
