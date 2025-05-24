# SSL encryption

Valid and trusted SSL certificates are needed to encrypt traffic between the client and server.  Certificates can be purchased online from various sources, or some organizations generate their own trusted certificates.  Regardless of which path you choose for enabling maximum security, the process to secure PMM consists of the following components:
{.power-number}

1. Staging the files in the proper locations:

    - You can [directly mount](#mounting-certificates) to a local directory containing the required certificates or
    - You can [copy the files](#copying-certificates) to the appropriate directory in your Container|AMI|OVF

2. Restarting PMM.
3. Ensuring the Clients trust the certificate issuer ([Ubuntu](https://ubuntu.com/server/docs/install-a-root-ca-certificate-in-the-trust-store) | [RedHat](https://www.redhat.com/sysadmin/configure-ca-trust-list) can get you started but this is somewhat OS specific)


With our Docker, OVF and AMI images, certificates are stored in `/srv/nginx` and our self-signed certificates are staged there by default.

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

For the new trusted certificates to take effect, you'll just need to restart the PMM Server (or advanced users can restart just nginx from a shell: supervisorctl restart nginx). 

You can now register clients to the PMM Server using the following:
```sh
pmm-admin config --server-url=https://<user>:<password>@<server IP>
```

!!! hint alert alert-success "Remember"
    Your client machine(s) must trust the issuer of the certificate, or you will still see "untrusted connections" messages when accessing the web interface. Thus, your client will need the `--server-insecure-tls` parameter when running the `pmm-admin config` command. Follow the instructions on your operating system to install the issuer certificate (ca-certs.pem). 

In case of PMM Client running in the container, mount certificates to `/etc/pki/tls/certs`:

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


