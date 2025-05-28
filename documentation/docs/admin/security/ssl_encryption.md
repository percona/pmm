# SSL encryption

Securing your PMM deployment with SSL/TLS encryption protects sensitive database metrics and authentication credentials in transit. 
You can obtain SSL certificates from several sources:

- commercial certificates from public CAs (automatically trusted by all systems)
- internal CA certificates from your organization 
- self-signed certificates for testing and development environments

## Prerequisites

Before implementing SSL encryption, ensure you have:

- valid SSL certificates (certificate, private key, and CA certificate)
- root or administrative access to configure certificates
- network connectivity between PMM Server and PMM Clients on HTTPS port (443)

## Requirements for setting up SSL certificates 

When configuring SSL certificates for PMM Client, keep in mind that the `SSL_CERT_FILE` environment variable affects **all** SSL/TLS connections made by the PMM Client process and ensure that:

- the CA bundle file is readable by the user running PMM Client: `chmod 644 /path/to/ca-bundle.pem`
- use the server's **hostname** (not IP address) in URLs to match the certificate

## Implement SSL encryption

To implement SSL encryption in PMM:
{.power-number}

1. **Prepare your certificates**: Choose one method to provide certificates to PMM Server:
   - [Mount certificates](#mounting-certificates) from a local directory on the host
   - [Copy certificates](#copying-certificates) directly into the PMM Server container

2. **Restart PMM Server** to apply the new certificates

3. **Configure client trust**: Choose your preferred method for PMM Client certificate verification:
   - [System trust store](#using-system-trust-store) (recommended for production)
   - [Custom CA certificates](#using-custom-ca-certificates) with `SSL_CERT_FILE` environment variable

## Location for certificate storage 

With Docker, OVF, and AMI deployments, certificates are stored in `/srv/nginx` where self-signed certificates are placed by default.

## Mount certificates

For container-based installation, mount your certificate directory to `/srv/nginx`:

```sh
docker run -d -p 443:443 --volumes-from pmm-data \
  --name pmm-server -v /etc/pmm-certs:/srv/nginx \
  --restart always percona/pmm-server:3
```

### Certificate requirements

- All certificates must be owned by root: `chown 0:0 /etc/pmm-certs/*`
- Set proper permissions: `chmod 644 /etc/pmm-certs/*.crt /etc/pmm-certs/*.pem && chmod 600 /etc/pmm-certs/*.key`
- The certificate directory must contain: `certificate.crt`, `certificate.key`, `ca-certs.pem`, and `dhparam.pem`
- Use port `443` for SSL encryption

## Copy certificates

If PMM Server is already running, copy certificates directly into the container:

```sh
# Copy certificate files
docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
docker cp certificate.key pmm-server:/srv/nginx/certificate.key
docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem
docker cp dhparam.pem pmm-server:/srv/nginx/dhparam.pem

# Set proper ownership and permissions
docker exec -it pmm-server chown root:root /srv/nginx/*
docker exec -it pmm-server chmod 644 /srv/nginx/*.crt /srv/nginx/*.pem
docker exec -it pmm-server chmod 600 /srv/nginx/*.key

# Restart to apply changes
docker restart pmm-server
```

## Connect PMM Client to PMM Server

After installing certificates and restarting PMM Server, register clients using HTTPS:

```sh
pmm-admin config --server-url=https://<user>:<password>@<server-hostname>
```

## Verify Client certificates

=== "Using system trust store"
    For production environments, install the CA certificate in your system's trust store:

    - **Ubuntu/Debian**: Follow the [Ubuntu CA certificate guide](https://ubuntu.com/server/docs/install-a-root-ca-certificate-in-the-trust-store)
    - **Red Hat/CentOS**: Follow the [Red Hat CA trust guide](https://www.redhat.com/sysadmin/configure-ca-trust-list)

    This method ensures all applications system-wide trust your certificates.

=== "Using custom CA certificates"
    For environments where you cannot or prefer not to modify the system's global certificate trust store, use the `SSL_CERT_FILE` environment variable. This approach is useful when:

    - PMM Server uses certificates signed by an internal/custom certificate authority
    - You want to avoid modifying system-wide certificate settings
    - Running in containerized or restricted environments
    - Testing with different CA configurations
    
    **Set SSL_CERT_FILE for `pmm-admin` commands

    Export the `SSL_CERT_FILE` environment variable pointing to your custom CA bundle file *before* running `pmm-admin` commands:

    ```sh
    export SSL_CERT_FILE=/path/to/custom-ca-bundle.pem
    pmm-admin config --server-url=https://<user>:<password>@<server-hostname>
    ```

#### Persistent configuration

For persistent configuration, add the environment variable to your shell profile or systemd service configuration:

=== "Shell profile"
    Add to your shell profile (.bashrc, .zshrc, etc.):
    ```sh         
    export SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem
    ```

=== "Systemd service"
    For PMM Client running as a systemd service:
    ```ini
    [Unit]
    Description=PMM Client
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
    When running PMM Client in a container:
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

### Create CA bundle files

The CA bundle file should contain one or more PEM-encoded certificate authority certificates:

```sh
# Single CA certificate
cp /path/to/your-ca.pem /etc/ssl/certs/custom-ca-bundle.pem

# Multiple CA certificates
cat /path/to/custom-ca1.pem /path/to/custom-ca2.pem > /etc/ssl/certs/custom-ca-bundle.pem

# Include system CAs plus custom CAs
cat /etc/ssl/certs/ca-certificates.crt \
    /path/to/your-custom-ca.pem > /etc/ssl/certs/custom-ca-bundle.pem
```

## Verify SSL configuration

Test your SSL configuration works correctly:

```sh
# Test certificate validation
export SSL_CERT_FILE=/etc/ssl/certs/custom-ca-bundle.pem
curl -v https://<server-hostname>/ping

# Test PMM Client connection
pmm-admin status
```