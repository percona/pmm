# PMM data encryption

Percona Monitoring and Management (PMM) implements robust encryption for sensitive data stored in its internal database, such as database access credentials, TLS certificates and keys, cloud credentials and backup location secrets.

## Default encryption

PMM automatically manages encryption using a keyset file located at `/srv/pmm-encryption.key`. PMM generates this file upon the initial launch of PMM 3 or when upgrading from the latest version of PMM 2.

Encrypted values are stored with the `pmm1$` prefix followed by the base64-encoded ciphertext (AES-256-GCM). The ciphertext embeds the ID of the key it was encrypted with, so PMM always knows which key of the keyset to use for decryption — including during key rotation.

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

Once configured, PMM will use the keyset to encrypt and decrypt all sensitive data stored within the system.

If the keyset file is unavailable or misplaced, PMM will be unable to access and decrypt the stored data, which will prevent it from running correctly.

Make sure to store and manage the encryption keyset securely to avoid potential loss of data access.

## Rotating the encryption key

You may want to rotate the encryption key when the original key is compromised or as part of routine security maintenance. For this, you can use the **PMM Encryption Rotation Tool**.

The tool adds a new key to the keyset and makes it the primary one; the previous keys remain in the keyset, so all stored data stays readable at every point of the rotation — the database is never held decrypted at rest. PMM Server is then restarted and re-encrypts all sensitive data with the new key during startup.

To rotate the encryption key:
{.power-number}

1. Log in to the container that runs PMM Server.

2. Run the Encryption Rotation Tool using the following command:

    ```bash
     pmm-encryption-rotation
    ```

    - Ensure `PMM_ENCRYPTION_KEY_PATH` is set to the current key file if using a custom location.
    - If using custom credentials/SSL for the PMM internal database, provide them with the appropriate flags.
    - Add `--prune` to remove the retired keys from the keyset once the tool has verified that no stored data references them anymore.

3. Verify PMM functionality all components are functioning properly to ensure that the encryption key rotation was successful.

Once the rotation tool has completed, the keyset file (at the default location `/srv/pmm-encryption.key` or the path specified by `PMM_ENCRYPTION_KEY_PATH`) contains the new primary key and all sensitive data is re-encrypted with it.

## Best practices for custom key management

- Always keep a secure backup of your encryption keyset, especially when using `PMM_ENCRYPTION_KEY_PATH`, as it is critical to PMM’s data decryption process.
- In containerized environments, ensure `PMM_ENCRYPTION_KEY_PATH` is persistently set in the container configuration to avoid issues during restarts.
- Test the encryption key rotation process in a staging environment before applying it in production to minimize potential downtime or configuration issues.
- Keep retired keys in the keyset (do not use `--prune`) until you have verified that the rotation completed successfully.

## See also

[Encrypt the PMM Client configuration file](client_config_encryption.md)
