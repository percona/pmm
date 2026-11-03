# PMM data encryption

Percona Monitoring and Management (PMM) encrypts the credentials it stores in its internal database: database access credentials, TLS client certificates and keys, cloud credentials and backup location secrets. See [What PMM encrypts](#what-pmm-encrypts) for the exact list.

## Default encryption

PMM automatically manages encryption using a keyset file located at `/srv/pmm-encryption.key`. PMM generates this file upon the initial launch of PMM 3 or when upgrading from the latest version of PMM 2.

Encrypted values are stored with the `pmm1$` prefix followed by the base64-encoded ciphertext (AES-256-GCM). The ciphertext embeds the ID of the key it was encrypted with, so PMM always knows which key of the keyset to use for decryption — including during key rotation.

## What PMM encrypts

PMM encrypts these fields of its internal database (`pmm-managed`):

| Data | Stored in | Encrypted fields |
|------|-----------|------------------|
| Service and agent credentials | `agents` | `username`, `password`, `agent_password` |
| Amazon RDS credentials | `agents.aws_options` | `aws_access_key`, `aws_secret_key` |
| Microsoft Azure credentials | `agents.azure_options` | `subscription_id`, `client_id`, `client_secret`, `tenant_id` |
| MongoDB TLS client key | `agents.mongo_options` | `tls_certificate_key`, `tls_certificate_key_file_password` |
| MySQL TLS client certificate and key | `agents.mysql_options` | `tls_cert`, `tls_key` |
| PostgreSQL TLS client certificate and key | `agents.postgresql_options` | `ssl_cert`, `ssl_key` |
| Valkey TLS client certificate and key | `agents.valkey_options` | `ssl_cert`, `ssl_key` |
| Backup location S3 credentials | `backup_locations.s3_config` | `access_key`, `secret_key` |

Not encrypted, because they are not secret:

- CA certificates (`tls_ca`, `ssl_ca`): they only verify the database server's certificate and are public.
- The SSH public key that can be set on AMI deployments: PMM writes it to the `authorized_keys` file, and it is public by design.
- Other settings and inventory data, such as addresses, ports, service names and labels.

Grafana stores its own secrets in its own database and encrypts them with its own key. To encrypt the PMM Client configuration file, see [Encrypt the PMM Client configuration file](client_config_encryption.md).

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
- keep retired keys in the keyset, without `--prune`, until you have verified a rotation.
- if PMM Server doesn't start because the key file is missing or was replaced, see [When PMM Server refuses to start](#when-pmm-server-refuses-to-start).

## HA clusters: use the same key on every node

In a [high availability (HA) cluster](../../install-pmm/install-HA-clustered.md), all PMM Server nodes share one PostgreSQL database, but each node reads the encryption key from its own local file. This means every node must use the **same** key.

A node with a different key can't decrypt the credentials that other nodes stored. PMM detects the mismatch and stops that node from starting. Otherwise, the node would send unusable credentials to PMM Clients, and monitoring would stop for those services.

The [PMM HA Helm chart](../../install-pmm/install-HA-clustered.md#manage-the-encryption-key) manages the shared key for you, including across upgrades and rescaling.

If you set up HA nodes yourself with the [HA variables](../../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#ha-variables):
{.power-number}

1. [Generate a single key](#set-up-a-custom-key) for the whole cluster.

2. Copy the key file to `/srv/pmm-encryption.key` on every node, or to the path set in `PMM_ENCRYPTION_KEY_PATH`.

3. Back up the key file together with the rest of your cluster configuration.

4. Start the nodes.

If a node has a different key, it logs that the encryption key `cannot decrypt stored credentials` and doesn't start. To fix it, copy the key file from a node that works to the same path on that node, then restart the node.

## Upgrading from PMM 3.9.1 or earlier

Earlier versions stored encrypted values without the `pmm1$` prefix, and some sensitive data (backup location S3 credentials, Valkey TLS certificates and keys) without encryption. The first start after the upgrade re-encrypts all of it in the new format with your existing key; the key file is used as is.

Before you upgrade:
{.power-number}

1. Back up the key file (`/srv/pmm-encryption.key` or the path in `PMM_ENCRYPTION_KEY_PATH`) together with your PMM data. A database backup cannot be read without it.

2. If PMM runs in [high availability mode](#ha-clusters-use-the-same-key-on-every-node), stop all PMM Server nodes, upgrade them, then start them. Nodes that still run the previous version cannot read data re-encrypted by upgraded nodes.

During the first start, PMM Server:

- Stores the values it is about to re-encrypt in `pmm-encryption-migration-backup-<timestamp>.json`, exactly as they were stored. The file goes next to the key file. If that directory is read-only, for example a key mounted from a secret, the file goes to `/srv`. If PMM Server can't write the file, it refuses to start and changes nothing. The file is readable by its owner only and is as sensitive as the database. Keep it until you have verified the upgrade, then delete it. If the first start is interrupted, the next one writes another backup file.
- Refuses to start if it cannot decrypt stored data, for example when the key file was replaced by a different key. The error lists the affected agents. Restore the original key file and restart; no data is changed. See [When PMM Server refuses to start](#when-pmm-server-refuses-to-start).
- Repairs credentials corrupted by key rotation in PMM 3.9.0 and earlier (see [below](#recover-from-a-corrupted-rotation)), using the previous key that the rotation left next to the key file (`pmm-encryption_old.key`, or `<name>_old.key` for a custom key path). Do not delete that file before upgrading.

!!! caution alert alert-warning "Downgrading is not supported"
    After the upgrade, earlier PMM versions cannot read the stored credentials. To go back, restore the PMM data backup taken before the upgrade.

## When PMM Server refuses to start

PMM Server checks at every start that its encryption key is the key that the stored data was encrypted with. It refuses to start, and changes nothing, in these cases:

- The key file is missing, but the database contains data encrypted with the key. PMM Server doesn't generate a new key, because that would make the stored data unreadable.
- The key file contains a different key. The error lists the affected agents. PMM Server also stores a known value encrypted with the key, so it detects a different key even before any credentials are stored.

In both cases, restore the original key file, or point `PMM_ENCRYPTION_KEY_PATH` at it, and restart PMM Server. In [high availability mode](#ha-clusters-use-the-same-key-on-every-node), copy the key file from a node that works.

### Recover from a lost encryption key

If the original key is permanently lost, you can start a standalone PMM Server without it. The credentials encrypted with the lost key can't be decrypted. PMM Server keeps them in a migration backup file, and you re-enter them afterward. This recovery isn't available in high availability mode, because the other nodes still hold the key.

To start PMM Server without the lost key:
{.power-number}

1. If the key file is missing, create a new key file in its place in the container that runs PMM Server:

    ```bash
    pmm-encryption-rotation --generate-key > /srv/pmm-encryption.key
    ```

    For a custom location, use the path in `PMM_ENCRYPTION_KEY_PATH`.

2. Restart PMM Server. It refuses to start, and `/srv/logs/pmm-managed.log` names the setting that accepts the loss for the current key, for example `PMM_ENCRYPTION_ACCEPT_KEY_LOSS=3749982648`.

3. Start PMM Server with that environment variable set to the value from the log. With Docker, re-create the container with the same data volume and add `-e PMM_ENCRYPTION_ACCEPT_KEY_LOSS=<ID from the log>`.

    PMM Server starts, keeps the values that it can't decrypt in `pmm-encryption-migration-backup-<timestamp>.json`, and logs the affected agents.

4. Re-enter the credentials of the affected services, or remove and re-add the services.

5. Remove `PMM_ENCRYPTION_ACCEPT_KEY_LOSS` from the container configuration. The setting accepts the loss only for the key that it names, so a key file that is replaced later is refused again.

If you find the lost key later, place it next to the key file as `pmm-encryption_old.key` (or `<name>_old.key` for a custom key path) and restart PMM Server. The credentials become readable again.

## Rotate the encryption key

Rotate the encryption key when it is compromised or as part of your regular security maintenance. The `pmm-encryption-rotation` tool generates a new key and re-encrypts all stored credentials with it.

The tool adds a new key to the keyset and makes it the primary key. The previous keys stay in the keyset, so all stored data stays readable during the rotation. The database is never held decrypted at rest. The tool then restarts PMM Server, which re-encrypts all encrypted fields with the new key during startup.

!!! note ""
    PMM Server stops briefly and restarts during rotation. Plan for a short monitoring gap.

### Prepare a custom setup

Skip this section if you use the default key path and default database settings.

#### Custom key path

- Make sure `PMM_ENCRYPTION_KEY_PATH` points to the current key, so the tool can decrypt existing data.
- Make sure the `pmm` user (UID 1000) can write to the directory that contains the key file. Otherwise, the rotation fails with a `permission denied` error before it changes anything. Fix the permissions and run it again.

#### Custom database credentials or SSL

- Have the credentials or SSL settings for the PMM internal database ready to pass as options to the rotation tool. To list the available options, run `pmm-encryption-rotation --help`.

### Rotate the key

To rotate the key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the rotation tool. If you use custom database credentials or SSL, add them as options:

    ```bash
    pmm-encryption-rotation
    ```

    Add `--prune` to remove the retired keys from the keyset once the tool has verified that no stored data references them anymore. Migration backup files are encrypted with keys that pruning removes. While such files exist, the tool refuses `--prune` unless you also add `--migration-backup-key-saved`. That option confirms that you keep a copy of the current key file elsewhere.

3. If you installed PMM Server with the [Helm chart](../../install-pmm/install-pmm-server/deployment-options/helm/index.md#manage-the-encryption-key), restart the pod so the chart updates its backup copy of the key:

    ```bash
    kubectl delete pod pmm-0
    ```

4. Confirm that PMM is working: check that your services appear as connected in **Inventory > Services** and that metrics are visible in your dashboards.

### Rotate the key on HA nodes

Key rotation isn't supported yet for the PMM HA Helm chart. The chart mounts the key from a read-only Kubernetes secret, which the rotation tool can't replace.

If you set up HA nodes yourself with the [HA variables](../../install-pmm/install-pmm-server/deployment-options/docker/env_var.md#ha-variables), all nodes must switch to the new key together. Rotate it on one node, then copy it to the others:
{.power-number}

1. Stop PMM Server on every node except one. A node left running can't read the values encrypted with the new key.

2. On the remaining node, [rotate the encryption key](#rotate-the-encryption-key) and add `--ha-other-nodes-stopped` to the command. Without it, the tool refuses to run in HA mode.

3. Copy the new key file from that node to every other node, using the same path.

4. Start the other nodes.


## Recover from a failed rotation

If PMM Server stops while `pmm-encryption-rotation` is running, your data remains intact. The keyset still holds the previous key, so all stored values stay readable.

To bring PMM back up and retry:
{.power-number}

1. Restart `pmm-managed`:

    ```bash
    supervisorctl start pmm-managed
    ```

2. If the directory containing your key file is not writable by the `pmm` user (UID 1000), fix the permissions:

    ```bash
    chown pmm:pmm /path/to/key/directory
    ```

3. Run the rotation tool again.

## Recover from a corrupted rotation

PMM versions before 3.9.1 contained a bug that corrupted certain credentials during key rotation: TLS certificates and keys and cloud credentials were encrypted more than once.

When you upgrade from such a version, PMM Server repairs these credentials automatically. It uses the previous key that the rotation left next to the key file. After more than one such rotation, the key of the innermost layer is no longer available. PMM Server then logs a warning during the upgrade that names the affected agents. If you have that key, place it at the `_old.key` path from the warning and restart PMM Server. Otherwise, re-add the services as described in [Corrupted credentials after encryption key rotation](../../troubleshoot/upgrade_issues.md#corrupted-credentials-after-encryption-key-rotation).


## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
