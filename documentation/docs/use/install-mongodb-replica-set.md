# Install a MongoDB replica set

!!! warning "Developer Preview"
    This feature is part of MongoDB Operations, a Developer Preview in PMM 3.10.0, intended for evaluation by Percona engineers and selected customers. It is not recommended for production use. The install workflow has known limitations and may not complete successfully in all configurations.

MongoDB Operations can install Percona Server for MongoDB on 1 or 3 empty nodes and register the replica set in PMM monitoring, without SSH access to the nodes.

## Prerequisites

Before you start, make sure that:

- MongoDB Operations is enabled. See [MongoDB Operations](mongodb-operations.md).
- Each node has run at least one successful scan. A scan runs automatically every 10 minutes; to run one immediately, use the scan action on the **Nodes** page.

## Node requirements

| Requirement | Scans | Installs |
|---|---|---|
| PMM Client 3.2.0 or later, installed from the DEB/RPM package | yes | yes |
| `pmm-agent` running as root (the default for the package's systemd service) | yes | yes |
| Writable cgroups and the `iproute` package | yes | yes |
| Outbound TCP to PMM Server on port 4647 | yes | yes |
| `python3` with the `venv` module (`python3-venv` on Ubuntu; included on Rocky Linux) | yes | yes |
| Access to PyPI or a configured pip mirror (the scan installs `pymongo`) | yes | yes |
| HTTPS to PMM Server's `/extensions` URL | no | yes |
| HTTPS to `repo.percona.com` and OS package mirrors | reported only | yes |
| Ubuntu or Rocky Linux, running systemd | no | yes |

### Node checks

PMM refuses a node for installation unless all of the following are true:

- The OS is **Ubuntu or Rocky Linux**, matched on `ID` in `/etc/os-release`. RHEL, AlmaLinux, Oracle Linux, and Debian are not supported. All nodes in one install must run the same OS.
- There is **no MongoDB on the node**: no `mongod` installed or running, no `/etc/mongod.conf`, and no MongoDB service registered for this node in PMM.
- The data directory (default `/var/lib/mongo`) is missing or empty and has **at least 5 GiB free**.
- Nothing else is listening on the chosen port (default 27017).
- The node is not the PMM Server host.
- **Node names resolve.** Replica set members address each other by their PMM node name, so every member must resolve every other member's node name, including its own, via DNS or `/etc/hosts`.
- The members can reach each other on the MongoDB port.

## Install a replica set

To install a MongoDB replica set:
{.power-number}

1. Go to **MongoDB > Operations > Nodes** and select **1 node** for a single-member replica set or **3 nodes** for a three-member replica set. Nodes that fail a check show the reason next to their name.

2. Click **Install MongoDB** and fill in the form:

    - **MongoDB version**: 7.0 (default) or 8.0, installed as Percona Server for MongoDB.
    - **Replica set name**.
    - **Data path** (default `/var/lib/mongo`), **log path** (default `/var/log/mongodb/mongod.log`), and **port** (default 27017).
    - **Bind address**: by default each member listens on its own address; `127.0.0.1` is always added.
    - Optional per-member settings: priority, votes, hidden, and delay for replica set elections.
    - Optional **environment** and **cluster** labels for PMM.
    - **Administrator user**: default name `admin` with the `root` role. Enter a password or use the generated one. The password must be at least 8 characters.

3. Review the settings and click to start the install.

    !!! warning ""
        A generated password is shown only once. PMM deletes its copy after creating the user. Save it before you close the confirmation screen.

4. Go to **Automations > Installs** to follow progress. Each node goes through these steps in order: pre-check, configure repository, install package, distribute keyfile, configure mongod, start service, and verify. After all nodes finish, PMM initiates the replica set, creates the users, and turns on authentication.

5. If a step fails, PMM retries it once, then rolls back what it installed on all nodes. Pre-check failures are not retried.

6. When the install finishes, the replica set appears in **Fleet** and is registered in PMM monitoring. The install takes a few minutes per node, mostly package download time.

## Security considerations

During installation, `mongod` starts without authentication until the final step enables it. This is intentional: enabling keyFile authentication before creating users fails on multi-member replica sets. The window is open to any host that can reach the node's MongoDB port. To reduce exposure, make sure the nodes are not reachable from untrusted networks during an install.

The bootstrap wizard sets up keyFile authentication with TLS off. There is no encryption at rest. These constraints apply to this Developer Preview only.

## Limitations

- Replica sets of exactly 1 or 3 members only. No sharded clusters or standalone installs.
- Ubuntu and Rocky Linux only.
- No TLS and no encryption at rest.
- Not for production use.
