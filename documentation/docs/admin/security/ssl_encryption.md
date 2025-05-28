# SSL encryption

Securing your PMM deployment with SSL/TLS encryption protects sensitive database metrics and authentication credentials in transit. This guide walks you through configuring SSL certificates for both PMM Server and PMM Clients.

You have several certificate options:
- **Commercial certificates** from public CAs (automatically trusted by all systems)
- **Internal CA certificates** from your organization's certificate authority  
- **Self-signed certificates** for testing and development environments

## Implementation steps

{.power-number}
1. **Prepare your certificates**: Choose one method to provide certificates to PMM Server:
   - [Mount certificates](#mounting-certificates) from a local directory on the host
   - [Copy certificates](#copying-certificates) directly into the PMM Server container

2. **Restart PMM Server** to apply the new certificates

3. **Configure client trust**: Ensure PMM Clients can verify the server certificate:
   - Add the CA certificate to the system trust store ([Ubuntu guide](https://ubuntu.com/server/docs/install-a-root-ca-certificate-in-the-trust-store) | [Red Hat guide](https://www.redhat.com/sysadmin/configure-ca-trust-list))
   - **Or use the `SSL_CERT_FILE` environment variable** for [custom CA certificates](#using-custom-ca-certificates-with-pmm-client)

!!! info "Certificate storage location"
    With Docker, OVF, and AMI deployments, certificates are stored in `/srv/nginx` where self-signed certificates are placed by default.

## Server-side certificate configuration

### Required certificate files

PMM Server requires these certificate files in `/srv/nginx`:

| File | Description |
|------|-------------|
| `certificate.crt` | Server certificate (may include intermediate certificates) |
| `certificate.key` | Private key for the server certificate |
| `ca-certs.pem` | Certificate Authority bundle |
| `dhparam.pem` | Diffie-Hellman parameters for enhanced security |

### Mounting certificates

For container-based installation, mount your certificate directory to `/srv/nginx`:

```sh
docker run -d -p 443:443 --volumes-from pmm-data \
  --name pmm-server -v /etc/pmm-certs:/srv/nginx \
  --restart always percona/pmm-server:3
```

!!! warning "Certificate requirements"
    - All certificates must be owned by root: `chown 0:0 /etc/pmm-certs/*`
    - Set proper permissions: `chmod 644 /etc/pmm-certs/*.crt /etc/pmm-certs/*.pem && chmod 600 /etc/pmm-certs/*.key`
    - The certificate directory must contain all four required files
    - Use port 443 for SSL encryption instead of port 80

### Copying certificates

If PMM Server is already running, copy certificates directly into the container:

```sh
# Copy certificate files
docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
docker cp certificate.key pmm-server:/srv/nginx/certificate.key
docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem
docker cp dhparam.pem pmm-server:/srv/nginx/dhparam.pem

# Set proper ownership
docker exec -it pmm-server chown root:root /srv/nginx/*
docker exec -it pmm-server chmod 644 /srv/nginx/*.crt /srv/nginx/*.pem
docker exec -it pmm-server chmod 600 /srv/nginx/*.key
```

### Applying certificate changes

Restart PMM Server to load the new certificates:

```sh
# Full container restart (recommended)
docker restart pmm-server

# Or restart only nginx (advanced users)
docker exec -it pmm-server supervisorctl restart nginx
```

Verify the certificates are working:
```sh
curl -I https://<server-hostname>:443
```

## Client-side certificate configuration

### Basic client connection

Register PMM Clients using the HTTPS URL:

```sh
pmm-admin config --server-url=https://<user>:<password>@<server-hostname>
```

!!! tip "Best practices"
    - Use the server's **hostname** (not IP address) to match the certificate
    - Ensure client system time is synchronized
    - Test the connection: `curl -I https://<server-hostname>`

### System-wide certificate trust

For production environments, install CA certificates in the system trust store:

=== "Ubuntu/Debian"
    ```sh
    # Install CA certificate
    sudo cp custom-ca.pem /usr/local/share/ca-certificates/custom-ca.crt
    sudo update-ca-certificates
    
    # Verify installation
    sudo update-ca-certificates --verbose
    ```

=== "Red Hat/CentOS/Fedora"
    ```sh
    # Install CA certificate  
    sudo cp custom-ca.pem /etc/pki/ca-trust/source/anchors/custom-ca.pem
    sudo update-ca-trust
    
    # Verify installation
    sudo update-ca-trust check
    ```

## Using custom CA certificates with PMM Client

For environments where you cannot modify the system's global certificate trust store, use the `SSL_CERT_FILE` environment variable. This approach is ideal when:

- PMM Server uses certificates signed by an internal/custom certificate authority
- You want to avoid modifying system-wide certificate settings  
- Running in containerized or restricted environments
- Testing with different CA configurations

### Setting SSL_CERT_FILE for pmm-admin commands

Export the `SSL_CERT_FILE` environment variable before running `pmm-admin` commands:

```sh
export SSL_CERT_FILE=/path/to/custom-ca-bundle.pem
pmm-admin config --server-url=https://<user>:<password>@<server-hostname>
```

### Persistent configuration

!!! note ""
    === "Shell profile"
        Add to your shell profile (.bashrc, .zshrc, etc.):
        ```sh         
        export SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem
        ```
        Then reload: `source ~/.bashrc`

    === "Systemd service"
        For pmm-agent running as a systemd service:
        ```ini
        [Unit]
        Description=PMM Agent
        After=network.target

        [Service]
        Type=simple
        User=pmm-agent
        Environment="SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem"
        ExecStart=/usr/local/bin/pmm-agent --config-file=/etc/pmm-agent.yaml
        Restart=always

        [Install]
        WantedBy=multi-user.target
        ```
        
    === "Docker container"
        When running pmm-client in a container:
        ```sh
        docker run \
          --rm --name pmm-client \
          -e PMM_AGENT_SERVER_ADDRESS=<server-hostname>:443 \
          -e PMM_AGENT_SERVER_USERNAME=admin \
          -e PMM_AGENT_SERVER_PASSWORD=admin \
          -e PMM_AGENT_SETUP=1 \
          -e SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem \
          -v /path/to/ca-bundle.pem:/etc/ssl/certs/custom-ca-bundle.pem:ro \
          percona/pmm-client:3
        ```

### Creating CA bundle files

The CA bundle file should contain one or more PEM-encoded certificate authority certificates:

```sh
# Single CA certificate
cp /path/to/your-ca.pem /etc/ssl/certs/custom-ca-bundle.pem

# Multiple CA certificates
cat /path/to/root-ca.pem \
    /path/to/intermediate-ca.pem > /etc/ssl/certs/custom-ca-bundle.pem

# Include system CAs plus custom CAs
cat /etc/ssl/certs/ca-certificates.crt \
    /path/to/your-custom-ca.pem > /etc/ssl/certs/custom-ca-bundle.pem
```

### Validation and troubleshooting

Verify your CA bundle and test the connection:

```sh
# Test certificate validation
openssl s_client -connect <server-hostname>:443 -CAfile /etc/ssl/certs/custom-ca-bundle.pem -verify_return_error

# Test pmm-agent connection
export SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem
curl -v https://<server-hostname>/ping
```

!!! warning "Important considerations"
    - The `SSL_CERT_FILE` environment variable affects **all** SSL/TLS connections made by the pmm-agent process
    - Ensure the CA bundle file is readable by the user running pmm-agent: `chmod 644 /path/to/ca-bundle.pem`
    - For debugging, test with curl before configuring pmm-admin

### Bypass certificate validation (testing only)

!!! danger "Security warning"
    **Only use for testing environments.** Never bypass certificate validation in production.

```sh
# Skip certificate validation (testing only)
pmm-admin config --server-url=https://<user>:<password>@<server-hostname> --server-insecure-tls
```