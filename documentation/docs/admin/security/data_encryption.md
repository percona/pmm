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

## Custom encryption key configuration

For enhanced security control, PMM supports a custom encryption keyset location.

**Key format requirements:**

- The file must contain a base64-encoded serialized Tink keyset with an AES-256-GCM key, as produced by `pmm-encryption-rotation --generate-key`.
- A raw 32-byte value is not a valid key file.

To generate a valid keyset file:

```bash
pmm-encryption-rotation --generate-key > /path/to/your/encryption.key
```

To set up a custom key location, configure the `PMM_ENCRYPTION_KEY_PATH` environment variable to point to your key file.

!!! hint alert alert-success "Important"
    Configure this **before** any data encryption occurs: either before upgrading to PMM 3 or before initially starting a new PMM 3.x instance.

### Key management requirements

Once configured, PMM uses the keyset to encrypt and decrypt the fields listed in [What PMM encrypts](#what-pmm-encrypts).

If the keyset file is unavailable or misplaced, PMM will be unable to access and decrypt the stored data, which will prevent it from running correctly.

Make sure to store and manage the encryption keyset securely to avoid potential loss of data access.

## Upgrading from PMM 3.9.1 or earlier

Earlier versions stored encrypted values without the `pmm1$` prefix, and some sensitive data (backup location S3 credentials, Valkey TLS certificates and keys) without encryption. The first start after the upgrade re-encrypts all of it in the new format with your existing key; the key file is used as is.

Before you upgrade:
{.power-number}

1. Back up the key file (`/srv/pmm-encryption.key` or the path in `PMM_ENCRYPTION_KEY_PATH`) together with your PMM data. A database backup cannot be read without it.

2. If PMM runs in [high availability mode](../../install-pmm/HA-docker.md), stop all PMM Server nodes, upgrade them, then start them. Nodes that still run the previous version cannot read data re-encrypted by upgraded nodes.

During the first start, PMM Server:

- Stores the values it is about to re-encrypt in `pmm-encryption-migration-backup-<timestamp>.json`, exactly as they were stored. The file goes next to the key file. If that directory is read-only, for example a key mounted from a secret, the file goes to `/srv`. If PMM Server can't write the file, it refuses to start and changes nothing. The file is readable by its owner only and is as sensitive as the database. Keep it until you have verified the upgrade, then delete it. If the first start is interrupted, the next one writes another backup file.
- Refuses to start if it cannot decrypt stored data, for example when the key file was replaced by a different key. The error lists the affected agents. Restore the original key file and restart; no data is changed. See [When PMM Server refuses to start](#when-pmm-server-refuses-to-start).
- Repairs credentials corrupted by key rotation in PMM 3.9.0 and earlier (see [below](#recovery-after-a-corrupted-rotation)), using the previous key that the rotation left next to the key file (`pmm-encryption_old.key`, or `<name>_old.key` for a custom key path). Do not delete that file before upgrading.

!!! caution alert alert-warning "Downgrading is not supported"
    After the upgrade, earlier PMM versions cannot read the stored credentials. To go back, restore the PMM data backup taken before the upgrade.

## When PMM Server refuses to start

PMM Server checks at every start that its encryption key is the key that the stored data was encrypted with. It refuses to start, and changes nothing, in these cases:

- The key file is missing, but the database contains data encrypted with the key. PMM Server doesn't generate a new key, because that would make the stored data unreadable.
- The key file contains a different key. The error lists the affected agents. PMM Server also stores a known value encrypted with the key, so it detects a different key even before any credentials are stored.

In both cases, restore the original key file, or point `PMM_ENCRYPTION_KEY_PATH` at it, and restart PMM Server. In [high availability mode](../../install-pmm/HA-docker.md), copy the key file from a node that works.

### Recover from a lost encryption key

If the original key is lost for good, you can start a standalone PMM Server without it. The credentials encrypted with the lost key can't be decrypted. PMM Server keeps them in a migration backup file, and you re-enter them afterwards. This recovery isn't available in high availability mode, because the other nodes still hold the key.

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

## Rotating the encryption key

You may want to rotate the encryption key when the original key is compromised or as part of routine security maintenance. For this, you can use the **PMM Encryption Rotation Tool**.

The tool adds a new key to the keyset and makes it the primary key. The previous keys stay in the keyset, so all stored data stays readable during the rotation. The database is never held decrypted at rest. The tool then restarts PMM Server, which re-encrypts all encrypted fields with the new key during startup.

To rotate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the Encryption Rotation Tool using the following command:

    ```bash
     pmm-encryption-rotation
    ```

    - Ensure `PMM_ENCRYPTION_KEY_PATH` is set to the current key file if using a custom location.
    - If using custom credentials/SSL for the PMM internal database, provide them with the appropriate flags.
    - Add `--prune` to remove the retired keys from the keyset once the tool has verified that no stored data references them anymore. Migration backup files can hold values encrypted with retired keys, so keep a copy of the key file from before the rotation while you need those files.

3. Verify PMM functionality all components are functioning properly to ensure that the encryption key rotation was successful.

Once the rotation tool has completed, the keyset file (at the default location `/srv/pmm-encryption.key` or the path specified by `PMM_ENCRYPTION_KEY_PATH`) contains the new primary key and all encrypted fields are re-encrypted with it.

### Rotate the key in high availability mode

In [high availability mode](../../install-pmm/HA-docker.md), the tool restarts PMM Server only on the node it runs on, and the other nodes can't read data encrypted with the new key. The tool therefore refuses to run until you confirm that PMM Server is stopped on every other node.

To rotate the key in high availability mode:
{.power-number}

1. Stop PMM Server on every node except one.

2. On the remaining node, run the Encryption Rotation Tool:

    ```bash
    pmm-encryption-rotation --ha-other-nodes-stopped
    ```

3. Copy the key file from that node to every other node, then start PMM Server on them.

## Recovery after a corrupted rotation

PMM versions before 3.9.1 contained a bug that corrupted certain credentials during key rotation: TLS certificates and keys and cloud credentials were encrypted more than once.

When you upgrade from such a version, PMM Server repairs these credentials automatically. It uses the previous key that the rotation left next to the key file. After more than one such rotation, the key of the innermost layer is no longer available. PMM Server then logs a warning during the upgrade that names the affected agents. If you have that key, place it at the `_old.key` path from the warning and restart PMM Server. Otherwise, re-add the services as described in [Corrupted credentials after encryption key rotation](../../troubleshoot/upgrade_issues.md#corrupted-credentials-after-encryption-key-rotation).

## Best practices for custom key management

- Always keep a secure backup of your encryption keyset, especially when using `PMM_ENCRYPTION_KEY_PATH`, as it is critical to PMM’s data decryption process.
- If PMM Server doesn't start because the key file is missing or was replaced, restore the original keyset. PMM Server doesn't generate a new key or re-encrypt data that it can't decrypt. No data is lost while you restore the original keyset. See [When PMM Server refuses to start](#when-pmm-server-refuses-to-start).
- In containerized environments, ensure `PMM_ENCRYPTION_KEY_PATH` is persistently set in the container configuration to avoid issues during restarts.
- Test the encryption key rotation process in a staging environment before applying it in production to minimize potential downtime or configuration issues.
- Keep retired keys in the keyset (do not use `--prune`) until you have verified that the rotation completed successfully.

## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
