# Isolated hosts

If the host where you will run [PMM Server][Docker image] has no internet connection, you can download the Docker image on a separate (internet-connected) host and securely copy it.
{.power-number}

1. On an internet-connected host, download the [Docker][Docker] image and its checksum file:

    ```sh
    wget https://downloads.percona.com/downloads/pmm/{{release}}/docker/pmm-server-{{release}}.docker
    wget https://downloads.percona.com/downloads/pmm/{{release}}/docker/pmm-server-{{release}}.sha256sum
    ```

2. Copy both files to where you will run PMM Server.

3. Open a terminal on the PMM Server host.

4. (Optional) Check the Docker image file integrity:

    ```sh
    shasum -ca 256 pmm-server-{{release}}.sha256sum
    ```

5. Load the image:

    ```sh
    docker load -i pmm-server-{{release}}.docker
    ```

6. [Run the container](index.md#run-docker-container) as if your image is already pulled using your desired method for a storage volume (you can step over any docker pull commands as the image has been pre-staged).

For more information, see:
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
[trusted certificate](../../../../admin/security/ssl_encryption.md)
[Easy-install script]: easy-install.md