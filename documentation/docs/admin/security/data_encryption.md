# PMM data encryption

Percona Monitoring and Management (PMM) encrypts the sensitive data it stores in its internal database, such as the access credentials and configuration details of monitored services.

## Default encryption

PMM automatically manages encryption using a key file located at `/srv/pmm-encryption.key`. PMM generates this file upon the initial launch of PMM 3 or when upgrading from the latest version of PMM 2.

## Use a custom encryption key

By default, PMM generates its own encryption key. To manage the key yourself, for example to keep it in your own secrets store, provide a custom key instead.

!!! hint alert alert-success "Important"
    Set up the custom key **before** PMM encrypts any data: before you upgrade to PMM 3, or before you start a new PMM 3 instance for the first time.

### Key format

If you provide your own key, it must be in the format PMM expects: a base64-encoded [Tink](https://developers.google.com/tink) keyset created from the `AES256GCMKeyTemplate`. A raw 32-byte value doesn't work, so don't create the key with a general-purpose tool such as `openssl rand`. If PMM can't read the key file, it doesn't start.

To get a key in the right format, generate it with the [Encryption Rotation Tool](#set-up-a-custom-key).

### Set up a custom key

To set up a custom key:
{.power-number}

1. Generate the key with `pmm-encryption-rotation`, a tool that comes with PMM Server, so there's nothing to install. Run it from the PMM Server Docker image:

    ```bash
    docker run --rm --entrypoint /usr/sbin/pmm-encryption-rotation percona/pmm-server:3 --generate-key > pmm-encryption.key
    ```

    This starts a temporary container, saves a new key to `pmm-encryption.key` in your current directory, and removes the container. The command doesn't connect to any database, so you can run it before you start PMM Server.

    If PMM Server is already running, you can instead run `pmm-encryption-rotation --generate-key > pmm-encryption.key` inside its container.

2. Make the key file available inside the PMM Server container, for example on a mounted volume.

3. Set the `PMM_ENCRYPTION_KEY_PATH` environment variable to the path of the key file inside the container.

### Keep the key safe

PMM uses the custom key to encrypt and decrypt all credentials it stores. If you lose the key, PMM can't decrypt those credentials. To avoid this:

- store the key securely and keep a backup of it outside PMM.
- in containerized environments, set `PMM_ENCRYPTION_KEY_PATH` in the container configuration so it persists across restarts.
- test key rotation in a staging environment before you rotate the key in production.

## HA clusters: use the same key on every node

In a [high availability (HA) cluster](../../install-pmm/install-HA-clustered.md), all PMM Server nodes share one PostgreSQL database, but each node reads the encryption key from its own local file. This means every node must use the **same** key.

A node with a different key can't decrypt the credentials that other nodes stored. PMM detects the mismatch and stops that node from starting. Otherwise, the node would send unusable credentials to PMM Clients, and monitoring would stop for those services.

If you deploy with the [PMM HA Helm chart](../../install-pmm/install-HA-clustered.md#manage-the-encryption-key), the chart manages the shared key for you. If you deploy with Docker:
{.power-number}

1. [Generate a single key](#set-up-a-custom-key) for the whole cluster.

2. Copy the key file to `/srv/pmm-encryption.key` on every node, or to the path set in `PMM_ENCRYPTION_KEY_PATH`.

3. Back up the key file together with the rest of your cluster configuration.

4. Start the nodes.

### Upgrade an existing Docker HA cluster

Before PMM 3.10.0, each node in a Docker HA cluster could generate its own key, and only one of these keys matches the credentials stored in the database. Starting with PMM 3.10.0, a node with a non-matching key doesn't start. If you plan to upgrade such a cluster to PMM 3.10.0, first make sure every node has the same key:
{.power-number}

1. Find the node where monitoring works. Its key file is the one that matches the database.

2. Copy that key file to every other node, using the same path.

3. Upgrade the nodes to PMM 3.10.0 and start them.

If a node still has a different key, it logs `encryption key does not match the database` and doesn't start. Copy the key from a node that works, then restart the node.

Clusters deployed with the PMM HA Helm chart need no action, because the chart already gives every node the same key.

## Rotate the encryption key

Rotate the encryption key if it's compromised, or as part of routine security maintenance. The `pmm-encryption-rotation` tool generates a new key and re-encrypts all stored credentials with it. While it runs, the tool briefly stops and restarts PMM Server.

To rotate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the rotation tool. If you use a custom key, first make sure `PMM_ENCRYPTION_KEY_PATH` points to the current key, so the tool can decrypt the existing data. If the PMM internal database uses custom credentials or SSL, add them as options to the command. To list the available options, run `pmm-encryption-rotation --help`:

```bash
    pmm-encryption-rotation
```

3. Check that PMM works as expected, for example that your services are still monitored.

The tool saves the new key to the same location as the old one: `/srv/pmm-encryption.key`, or the path set in `PMM_ENCRYPTION_KEY_PATH`.

If you installed PMM Server with the [Helm chart](../../install-pmm/install-pmm-server/deployment-options/helm/index.md#manage-the-encryption-key), restart the pod after rotation, for example with `kubectl delete pod pmm-0`. The chart updates its backup copy of the key only when the pod starts.

### Rotate the key in a Docker HA cluster

All nodes must switch to the new key together, so rotate it on one node and then copy it to the others:
{.power-number}

1. Stop PMM Server on every node except one. Any node left running keeps using the old key, which leaves the database encrypted with two different keys.

2. On the remaining node, [rotate the encryption key](#rotate-the-encryption-key).

3. Copy the new key file from that node to every other node, using the same path.

4. Start the other nodes.

Key rotation isn't supported yet for the PMM HA Helm chart. The chart mounts the key from a read-only Kubernetes secret, which the rotation tool can't replace.


## Recover from a corrupted rotation

PMM versions before 3.9.1 contained a bug that corrupted certain credentials during key rotation. If you rotated the encryption key before upgrading to 3.9.1, see [Corrupted credentials after encryption key rotation](../../troubleshoot/upgrade_issues.md#corrupted-credentials-after-encryption-key-rotation) for recovery steps.


## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
