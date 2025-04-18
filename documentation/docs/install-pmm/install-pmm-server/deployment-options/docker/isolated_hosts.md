# Install PMM Server in isolated environments

To deploy [PMM Server][Docker image] in air-gapped or isolated environments with no direct internet connectivity, download the Docker image on a separate (internet-connected) host and securely copy it:
{.power-number}

1. On an internet-connected host, download the [Docker][Docker] image and its checksum file:

    ```sh
    wget https://downloads.percona.com/downloads/pmm/{{release}}/docker/pmm-server-{{release}}.docker
    wget https://downloads.percona.com/downloads/pmm/{{release}}/docker/pmm-server-{{release}}.sha256sum
    ```

2. Transfer both files to the target host where you'll run PMM Server using a secure method (such as `scp`, physical media, or your organization's approved file transfer mechanism).


3. On the target host, open a terminal and navigate to where you placed the downloaded files.

4. Verify the Docker image file integrity (recommended):

    ```sh
    shasum -ca 256 pmm-server-{{release}}.sha256sum
    ```

5. Load the Docker image:

    ```sh
    docker load -i pmm-server-{{release}}.docker
    ```

6. [Run the PMM Server container](index.md#run-docker-container) as if your image is already pulled using your desired method for a storage volume. Skip any `docker pull` commands as the image has been pre-staged and available locally.


## Related resources

- [Docker installation guide][Docker]
- [Docker Compose installation][Docker compose]
- [PMM Server Docker tags][tags]
- [PMM Client Docker setup][PMMC_COMPOSE]
- [Setting up trusted certificates][trusted certificate]
- [Easy installation script][Easy-install script]

[tags]: https://hub.docker.com/r/percona/pmm-server/tags
[Docker]: https://docs.docker.com/get-docker/
[Docker image]: https://hub.docker.com/r/percona/pmm-server
[Docker compose]: https://docs.docker.com/compose/
[PMMC_COMPOSE]: ../../../install-pmm-client/docker.md
[trusted certificate]: ../../../../admin/security/ssl_encryption.md
[Easy-install script]: easy-install.md     