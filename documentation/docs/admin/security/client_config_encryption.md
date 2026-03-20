# PMM Client configuration file encryption

The PMM Client configuration file, `pmm-agent.yaml`(../../use/commands/pmm-agent.md) contains sensitive information like PMM Server credentials and API tokens. By default, this file is stored in plain text, which means that users with read access to the filesystem can see these credentials.

To protect this data, you can encrypt the configuration file so that its contents are unreadable on disk. 

This involves generating an RSA private key and passing it to PMM Client during setup. PMM then automatically encrypts the file whenever it saves configuration changes and decrypts it at startup.

Encryption is optional. Without an encryption key, PMM Client continues to read and write the configuration file in plain text.

## Before you start

To encrypt the PMM Client configuration file, you need:

- **PMM Client 3.7.0** or later
- **[OpenSSL](https://docs.openssl.org/master/man1/openssl-genpkey/)** (or any compatible tool) to generate an RSA private key

## How it works

PMM Client uses two layers of encryption to protect the configuration file:

- **AES-256-GCM** encrypts the configuration data and guards against tampering.
- **RSA-OAEP** wraps the AES key so that only your RSA private key can unlock it.

For additional security, you can also protect the RSA private key with a password.

## Set up encryption
To encrypt the PMM Client configuration file, generate an RSA private key and pass it to PMM Client during setup and at startup:
{.power-number}

1. Generate an RSA private key:

    === "Password-protected (recommended)"
        ```bash
        openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
          -aes256 -pass env:OPENSSL_PASSWORD \
          -out /etc/pmm-agent-key.pem
        ```

    === "Without password"
        ```bash
        openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
          -out /etc/pmm-agent-key.pem
        ```
2. Set permissions on the key file:

    ```bash
    chmod 600 /etc/pmm-agent-key.pem
    chown pmm-agent:pmm-agent /etc/pmm-agent-key.pem
    ```

3. Run `pmm-agent setup` with the encryption flags:

    ```bash
    pmm-agent setup \
      --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml \
      --server-address=pmm-server.example.com:443 \
      --server-insecure-tls \
      --config-file-key-file=/etc/pmm-agent-key.pem \
      --config-file-key-password="$OPENSSL_PASSWORD" \
      --server-username=admin \
      --server-password=admin
    ```

    If your key is not password-protected, omit `--config-file-key-password`.

4. Start PMM Client with the encryption flags:

    ```bash
    pmm-agent --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml \
      --config-file-key-file=/etc/pmm-agent-key.pem \
      --config-file-key-password="$OPENSSL_PASSWORD"
    ```

## Encryption settings

PMM Client accepts encryption settings as either command-line flags or environment variables. Use flags when running `pmm-agent` directly, and environment variables when configuring a service manager like systemd, Docker, or Kubernetes.


| Flag | Environment variable | Description |
|------|---------------------|-------------|
| `--config-file-key-file` | `PMM_AGENT_CONFIG_FILE_KEY_FILE` | Path to the RSA private key file. Required to enable encryption. |
| `--config-file-key-password` | `PMM_AGENT_CONFIG_FILE_KEY_PASSWORD` | Password for the RSA private key. Only needed if the key is password-protected. | 

| Flag | Environment variable | Description |
|------|---------------------|-------------|
| `--config-file-key-file` | `PMM_AGENT_CONFIG_FILE_KEY_FILE` | Path to the RSA private key file. Required to enable encryption. |
| `--config-file-key-password` | `PMM_AGENT_CONFIG_FILE_KEY_PASSWORD` | Password for the RSA private key. Only needed if the key is password-protected. |

## Deployment examples

=== "systemd"   

    Create or modify `/etc/systemd/system/pmm-agent.service`:

    ```ini
    [Unit]
    Description=PMM Agent
    After=network.target

    [Service]
    Type=simple
    User=pmm-agent
    Group=pmm-agent

    Environment="PMM_AGENT_CONFIG_FILE_KEY_FILE=/etc/pmm-agent-key.pem"
    # For password-protected keys, use one of the following:
    # Option 1: systemd credentials (systemd 247+)
    # LoadCredential=key_password:/etc/pmm-agent-key-password
    # Option 2: Environment file with restricted permissions
    # EnvironmentFile=-/etc/pmm-agent-encryption.env

    ExecStart=/usr/local/percona/pmm/bin/pmm-agent \
    --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml

    Restart=on-failure
    RestartSec=10s

    [Install]
    WantedBy=multi-user.target
    ```

=== "Docker/Podman"

    ```bash
    docker run -d \
    --name pmm-agent \
    -v /usr/local/percona/pmm/config/pmm-agent.yaml:/usr/local/percona/pmm/config/pmm-agent.yaml \
    -v /etc/pmm-agent-key.pem:/etc/pmm-agent-key.pem:ro \
    -e PMM_AGENT_CONFIG_FILE_KEY_FILE=/etc/pmm-agent-key.pem \
    -e PMM_AGENT_CONFIG_FILE_KEY_PASSWORD=your-password \
    percona/pmm-client:3 \
    --config-file=/usr/local/percona/pmm/config/pmm-agent.yaml
    ```

=== "Kubernetes" 

    ```yaml
    apiVersion: v1
    kind: Secret
    metadata:
    name: pmm-agent-encryption-key
    type: Opaque
    data:
    key.pem: <base64-encoded-RSA-key>
    key-password: <base64-encoded-password>
    ---
    apiVersion: apps/v1
    kind: Deployment
    metadata:
    name: pmm-agent
    spec:
    template:
        spec:
        containers:
        - name: pmm-agent
            image: percona/pmm-client:3
            env:
            - name: PMM_AGENT_CONFIG_FILE_KEY_FILE
            value: /etc/encryption/key.pem
            - name: PMM_AGENT_CONFIG_FILE_KEY_PASSWORD
            valueFrom:
                secretKeyRef:
                name: pmm-agent-encryption-key
                key: key-password
            volumeMounts:
            - name: encryption-key
            mountPath: /etc/encryption
            readOnly: true
            - name: config
            mountPath: /usr/local/percona/pmm/config/pmm-agent.yaml
            subPath: pmm-agent.yaml
        volumes:
        - name: encryption-key
            secret:
            secretName: pmm-agent-encryption-key
            defaultMode: 0600
        - name: config
            persistentVolumeClaim:
            claimName: pmm-agent-config
    ```

## Migrate from an unencrypted configuration

If PMM Client is already set up, you can enable encryption without re-registering the agent:
{.power-number}

1. Generate an encryption key:

    ```bash
    openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:4096 \
      -aes256 -pass env:OPENSSL_PASSWORD \
      -out /etc/pmm-agent-key.pem
    chmod 600 /etc/pmm-agent-key.pem
    ```

2. Stop PMM Client:

    ```bash
    systemctl stop pmm-agent
    ```

3. Add the encryption environment variables to your [systemd, Docker, or Kubernetes configuration](#deployment-examples).

4. Restart PMM Client to apply the new encryption settings:
    ```bash
    systemctl start pmm-agent
    ```

PMM Client automatically encrypts the configuration file on the next save.

## Disable encryption

To remove encryption and store the configuration file in plain text:
{.power-number}

1. Remove the encryption environment variables (`PMM_AGENT_CONFIG_FILE_KEY_FILE` and `PMM_AGENT_CONFIG_FILE_KEY_PASSWORD`) from your [systemd, Docker, or Kubernetes configuration](#deployment-examples).
2. Restart PMM Client so it can decrypt the file and rewrite it in plain text while the key is still in memory. If you skip this step, the file remains encrypted and PMM Client won't be able to read it on future restarts.

## Verify encryption status

Check whether a configuration file is encrypted by reading it directly:

```bash
# Encrypted: shows binary content, not valid YAML
cat /usr/local/percona/pmm/config/pmm-agent.yaml

# You can also confirm with hexdump
head -c 100 /usr/local/percona/pmm/config/pmm-agent.yaml | hexdump -C
```

A plain-text file shows readable YAML. An encrypted file shows binary data.

## Key management best practices

- **Back up keys** separately from encrypted configuration files.
- **Use password protection** for RSA private keys.
- **Restrict file permissions** to `0600`, owned by the `pmm-agent` user.
- **Store keys and configuration files in different locations** when possible.
- **Use a secret management system** (HashiCorp Vault, AWS Secrets Manager, etc.) in production environments.
- **Implement key rotation** based on your compliance requirements.
- **Avoid embedding passwords** in scripts or configuration files. Use environment files with restricted permissions or systemd credentials instead.

## Troubleshooting

### "unable to get RSA key from KeyFile"

- Check that the key file path is correct.
- Verify file permissions (must be readable by the `pmm-agent` user).
- Confirm the file contains a valid RSA private key in PEM format.

### "pkcs8: incorrect password"

- verify the password is correct.
- check that `PMM_AGENT_CONFIG_FILE_KEY_PASSWORD` (or `--config-file-key-password`) is set correctly.

### "unable to RSA-unwrap AES key: crypto/rsa: decryption error"

The configuration file was encrypted with a different key, or the file may be corrupted. Restore from backup or regenerate the configuration.

### "no valid private key found in a KeyFile"

The key file is not in the correct PEM format or may be corrupted. Regenerate the key file.

## Technical specifications

If you need cryptographic details for security audits or compliance reviews:

- **Encryption**: AES-256-GCM (Galois/Counter Mode), 32-byte key, 12-byte nonce
- **Key wrapping**: RSA-OAEP with SHA-256
- **RSA key size**: 2048 bits minimum, 4096 recommended
- **Key format**: PKCS#8 PEM