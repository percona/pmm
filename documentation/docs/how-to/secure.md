# Secure

You can improve the security of your PMM installation with:

- [SSL encryption](#ssl-encryption) to secure traffic between client and server;

- [Grafana HTTPS secure cookies](#grafana-https-secure-cookies)

To see which security features are enabled:

```sh
pmm-admin status
```

!!! hint alert alert-success "Tip"
    You can gain an extra level of security by keeping PMM Server isolated from the internet, if possible.

## SSL encryption

You need valid SSL certificates to encrypt traffic between client and server.

With our Docker, OVF and AMI images, self-signed certificates are in `/srv/nginx`.

To use your own, you can either:

- mount the local certificate directory to the same location, or,

- copy your certificates to a running PMM Server container.

### Mounting certificates

For example, if your own certificates are in `/etc/pmm-certs`:

```sh
docker run -d -p 443:8443 --name pmm-server  \
  -v pmm-data:/srv -v /etc/pmm-certs:/srv/nginx \
  --restart always percona/pmm-server:3
```

!!! note alert alert-primary ""
    - The certificates must be owned by root. You can do this with: `chown 0:0 /etc/pmm-certs/*`
    - The mounted certificate directory (`/etc/pmm-certs` in this example) must contain the files `certificate.crt`, `certificate.key`, `ca-certs.pem` and `dhparam.pem`.
    - For SSL encryption, the container must expose port 443 instead of 80.

### Copying certificates

If PMM Server is running as a Docker image, use `docker cp` to copy certificates. This example copies certificate files from the current working directory to a running PMM Server docker container.

```sh
docker cp certificate.crt pmm-server:/srv/nginx/certificate.crt
docker cp certificate.key pmm-server:/srv/nginx/certificate.key
docker cp ca-certs.pem pmm-server:/srv/nginx/ca-certs.pem
docker cp dhparam.pem pmm-server:/srv/nginx/dhparam.pem
```

### Enabling SSL when connecting PMM Client to PMM Server

```sh
pmm-admin config --server-url=https://<user>:<password>@<server IP>
```

## Grafana HTTPS secure cookies

To enable:

1. Start a shell within the Docker container.

    ```sh
    docker exec -it pmm-server bash
    ```

2. Edit `/etc/grafana/grafana.ini`.

3. Enable `cookie_secure` and set the value to `true`.

4. Restart Grafana.

    ```sh
    supervisorctl restart grafana
    ```
