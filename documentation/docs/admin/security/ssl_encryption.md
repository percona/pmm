# SSL encryption

SSL encryption protects communication between PMM Client and PMM Server by encrypting all data in transit. To enable SSL, you need valid certificates that can be:

- **Purchased certificates**: Obtained from commercial Certificate Authorities (CAs) like Let's Encrypt, DigiCert, or GlobalSign
- **Self-signed certificates**: Generated internally for testing or closed environments
- **Internal CA certificates**: Created by your organization's internal Certificate Authority

To implement SSL encryption in PMM:
{.power-number}

1. **Prepare your certificates**: Choose one method to provide certificates to PMM Server:
   - [Mount certificates](#mounting-certificates) from a local directory on the host
   - [Copy certificates](#copying-certificates) directly into the PMM Server container

2. **Restart PMM Server** to apply the new certificates.

3. **Configure client trust**: Ensure PMM Clients can verify the server certificate:
   - Add the CA certificate to the system trust store ([Ubuntu guide](https://ubuntu.com/server/docs/install-a-root-ca-certificate-in-the-trust-store) | [Red Hat guide](https://www.redhat.com/sysadmin/configure-ca-trust-list))
   - **Or use the `SSL_CERT_FILE` environment variable** for [custom CA certificates](#using-custom-ca-certificates-with-pmm-client).

With Docker, OVF, and AMI deployments, certificates are stored in `/srv/nginx` where self-signed certificates are placed by default.

### Mounting certificates

For container-based installation, if your certificates are in a directory called `/etc/pmm-certs` on the container host, run the following to mount that directory in the proper location so that PMM can find it when the container starts:

```sh
docker run -d -p 443:443 --volumes-from pmm-data \
  --name pmm-server -v /etc/pmm-certs:/srv/nginx \
  --restart always percona/pmm-server:3
```

!!! note alert alert-primary ""
    - All certificates must be owned by root. You can do this with: `chown 0:0 /etc/pmm-certs/*`
    - The mounted certificate directory (`/etc/pmm-certs` in this example) must contain the files named `certificate.crt`, `certificate.key`, `ca-certs.pem`, and `dhparam.pem`.
    - For SSL encryption, the container should publish on port 443 instead of 80.

### Copying certificates

If PMM Server is running as a Docker image, use `docker cp` to copy certificates. This example copies certificate files from the current working directory to a running PMM Server docker container.

```sh
docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
docker cp certificate.key pmm-server:/srv/nginx/certificate.key
docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem
docker cp dhparam.pem pmm-server:/srv/nginx/dhparam.pem
docker exec -it pmm-server chown pmm:pmm /srv/nginx/*
```

### Use trusted SSL when connecting PMM Client to PMM Server

To apply the new trusted certificates, simply restart the PMM Server. If you're an advanced user, you can alternatively restart only the NGINX service from the shell using: `supervisorctl restart nginx`.

You can now register clients to the PMM Server using the following:
```sh
pmm-admin config --server-url=https://<user>:<password>@<server IP>
```

!!! hint alert alert-success "Remember"
    Your client machine(s) must trust the issuer of the certificate, or you will still see "untrusted connections" messages when accessing the web interface. Thus, your client will need the `--server-insecure-tls` parameter when running the `pmm-admin config` command. Follow the instructions on your operating system to install the issuer certificate (ca-certs.pem). 

### Using custom CA certificates with PMM Client

If your PMM Server uses a certificate signed by a custom Certificate Authority (CA) that is not trusted by the host running `pmm-agent`, you can configure the client to trust the custom CA using the `SSL_CERT_FILE` environment variable:

=== "Direct host installation"

    For PMM Client installed directly on the host system:

    Set the `SSL_CERT_FILE` environment variable to point to your custom CA certificate file:

    ```sh
    export SSL_CERT_FILE=/path/to/your/ca-certificate.pem
    pmm-admin config --server-url=https://<user>:<password>@<server IP>
    ```

    !!! note "Persistent environment variable"
        To make the SSL_CERT_FILE setting persistent across reboots, add it to your shell profile:
        ```sh
        echo 'export SSL_CERT_FILE=/path/to/your/ca-certificate.pem' >> ~/.bashrc
        source ~/.bashrc
        ```

=== "Container deployment"

    For PMM Client running as a Docker container:

    **Option 1: Using SSL_CERT_FILE environment variable**

    Mount your custom CA certificate file and set the `SSL_CERT_FILE` environment variable:

    ```sh
    PMM_SERVER=X.X.X.X:443
    docker run \
    --rm \
    --name pmm-client \
    -e PMM_AGENT_SERVER_ADDRESS=${PMM_SERVER} \
    -e PMM_AGENT_SERVER_USERNAME=admin \
    -e PMM_AGENT_SERVER_PASSWORD=admin \
    -e PMM_AGENT_SETUP=1 \
    -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
    -e SSL_CERT_FILE=/etc/ssl/certs/custom-ca.pem \
    -v /path/to/your/custom-ca.pem:/etc/ssl/certs/custom-ca.pem \
    percona/pmm-client:3
    ```

    **Option 2: Mount to standard certificate directory**

    Mount certificates to the standard certificate directory `/etc/pki/tls/certs`:

    ```sh
    PMM_SERVER=X.X.X.X:443
    docker run \
    --rm \
    --name pmm-client \
    -e PMM_AGENT_SERVER_ADDRESS=${PMM_SERVER} \
    -e PMM_AGENT_SERVER_USERNAME=admin \
    -e PMM_AGENT_SERVER_PASSWORD=admin \
    -e PMM_AGENT_SETUP=1 \
    -e PMM_AGENT_CONFIG_FILE=config/pmm-agent.yaml \
    -v /your_directory_with/certs:/etc/pki/tls/certs \
    percona/pmm-client:3
    ```

    !!! tip "Container certificate management"
        When using containers, Option 1 with `SSL_CERT_FILE` gives you more precise control over which CA certificate is used, while Option 2 automatically trusts all certificates in the mounted directory.

!!! info "When to use SSL_CERT_FILE"
    The `SSL_CERT_FILE` environment variable is particularly useful when:
    
    - Your organization uses internal CA certificates
    - You cannot or don't want to modify the system's certificate trust store
    - You need a quick solution for testing with custom certificates
    - You want to specify a single CA file instead of adding it to the system-wide certificate store

