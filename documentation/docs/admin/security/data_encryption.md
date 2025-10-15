# PMM data encryption

Percona Monitoring and Management (PMM) implements robust encryption for sensitive data stored in its internal database's `agent` table. This includes access credentials and configuration details.

## Default encryption

PMM automatically manages encryption using a key file located at `/srv/pmm-encryption.key`. PMM generates this file upon the initial launch of PMM 3 or when upgrading from the latest version of PMM 2.

## Custom encryption key configuration

For enhanced security control, PMM supports custom encryption keys.

To set up a custom keys, configure the `PMM_ENCRYPTION_KEY_PATH` environment variable to point to your custom key file.

!!! hint alert alert-success "Important"
    Make sure to set this configuration  **before** any data encryption occurs—specifically, either before upgrading to PMM 3 or before the initial startup of a new PMM 3.x container.

### Key management requirements

Once configured, PMM will use custom keys to encrypt and decrypt all sensitive data stored within the system.

If the custom key is unavailable or misplaced, PMM will be unable to access and decrypt the stored data, which will prevent it from running correctly.

Make sure to store and manage the custom encryption key securely to avoid potential loss of data access.

## Rotating the encryption key

You may want to change or update the encryption key when the original key is compromised or as part of routine security maintenance. For this, you can use the **PMM Encryption Rotation Tool**.

This tool re-encrypts all existing sensitive data with a newly generated encryption key, ensuring continuous security with minimal disruption.

To rotate or regenerate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the Encryption Rotation Tool using the following the command:

    ```bash
   pmm-encryption-rotation
    ```

      - Ensure `PMM_ENCRYPTION_KEY_PATH` is set to the current custom key if using one, so the tool can decrypt data before re-encryption.
      - If using custom credentials/SSL for the PMM internal database, provide them with the appropriate flags.

3. Verify PMM functionality all components are functioning properly to ensure that the encryption key rotation was successful.

Once the rotation tool has completed, a new encryption key will be generated and saved either in the default location (`/srv/pmm-encryption.key`) or in the path specified by `PMM_ENCRYPTION_KEY_PATH`. The tool will automatically re-encrypt all sensitive data with the new key.

## Best pracices for custom key management

- Always keep a secure backup of your encryption key, especially when using `PMM_ENCRYPTION_KEY_PATH`, as it is critical to PMM’s data decryption process.
- In containerized environments, ensure `PMM_ENCRYPTION_KEY_PATH` is persistently set in the container configuration to avoid issues during restarts.
- Test the encryption key rotation process in a staging environment before applying it in production to minimize potential downtime or configuration issues.
