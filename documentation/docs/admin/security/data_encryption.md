# PMM data encryption

Percona Monitoring and Management (PMM) implements robust encryption for sensitive data stored in its internal database's `agent` table. This includes access credentials and configuration details.

## Default encryption

PMM automatically manages encryption using a key file located at `/srv/pmm-encryption.key`. PMM generates this file upon the initial launch of PMM 3 or when upgrading from the latest version of PMM 2.

## Custom encryption key configuration

For enhanced security control, PMM supports custom encryption keys.

**Key format requirements:**

The key file must contain a base64-encoded Tink keyset created from the `AES256GCMKeyTemplate`. This is not a raw 32-byte value, so a key produced with a general-purpose tool such as `openssl rand` cannot be used: PMM fails to start if it cannot parse the keyset.

Generate a key in the correct format with the Encryption Rotation Tool, which prints a new key to stdout without touching the database:

```bash
pmm-encryption-rotation --generate-key
```

To set up a custom key, write that value to a file and point the `PMM_ENCRYPTION_KEY_PATH` environment variable at it.

!!! hint alert alert-success "Important"
    Configure this **before** any data encryption occurs: either before upgrading to PMM 3 or before initially starting a new PMM 3.x instance.

### High availability deployments

All PMM Server nodes in a [highly available deployment](../../install-pmm/install-HA-clustered.md) share one PostgreSQL database, but each node reads its encryption key from its own local file. Every node must therefore use the **same** encryption key.

A node holding a different key cannot decrypt credentials written by the other nodes. PMM detects this and refuses to start the affected node, because otherwise it would hand unusable credentials to PMM Clients and monitoring would stop for the affected services.

Generate the key once, place it on every node before starting them, and back it up with the rest of your cluster configuration:

```bash
pmm-encryption-rotation --generate-key > pmm-encryption.key
```

To rotate the key in an HA cluster, run the [rotation procedure](#rotating-the-encryption-key) on a single node, then copy the resulting key file to all the other nodes and restart them.

### Key management requirements

Once configured, PMM will use the custom key to encrypt and decrypt all sensitive data stored within the system.

If the custom key is unavailable or misplaced, PMM will be unable to access and decrypt the stored data, which will prevent it from running correctly.

Make sure to store and manage the custom encryption key securely to avoid potential loss of data access.

## Rotating the encryption key

You may want to generate a new encryption key or rotate it when the original key is compromised or as part of routine security maintenance. For this, you can use the **PMM Encryption Rotation Tool**.

This tool re-encrypts all existing sensitive data with a newly generated encryption key, ensuring continuous security with minimal disruption.

To rotate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the Encryption Rotation Tool using the following command:

    ```bash
     pmm-encryption-rotation
    ```

    - Ensure `PMM_ENCRYPTION_KEY_PATH` is set to the current custom key if using one, so the tool can decrypt data before re-encryption.
    - If using custom credentials/SSL for the PMM internal database, provide them with the appropriate flags.

3. Verify PMM functionality all components are functioning properly to ensure that the encryption key rotation was successful.

Once the rotation tool has completed, a new encryption key will be generated and saved either in the default location (`/srv/pmm-encryption.key`) or in the path specified by `PMM_ENCRYPTION_KEY_PATH`. The tool will automatically re-encrypt all sensitive data with the new key.

## Recovery after a corrupted rotation

PMM versions before 3.9.1 contained a bug that corrupted certain credentials during key rotation. If you rotated the encryption key before upgrading to 3.9.1, see [Corrupted credentials after encryption key rotation](../../troubleshoot/upgrade_issues.md#corrupted-credentials-after-encryption-key-rotation) for recovery steps.

## Best practices for custom key management

- Always keep a secure backup of your encryption key, especially when using `PMM_ENCRYPTION_KEY_PATH`, as it is critical to PMM’s data decryption process.
- In containerized environments, ensure `PMM_ENCRYPTION_KEY_PATH` is persistently set in the container configuration to avoid issues during restarts.
- Test the encryption key rotation process in a staging environment before applying it in production to minimize potential downtime or configuration issues.

## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
