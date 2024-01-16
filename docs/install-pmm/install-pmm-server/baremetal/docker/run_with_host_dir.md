
# Run Docker with the host directory

!!! note alert alert-primary "Availability"
    This feature is available starting with PMM 2.29.0.

To run Docker with the host directory:
{.power-number}

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Run the image.

    ```sh
    export DATA_DIR=$HOME/srv
    docker run -v $DATA_DIR/srv:/srv -d --restart always --publish 80:80 --publish 443:443 --name pmm-server percona/pmm-server:2
    ```
    
    `DATA_DIR` is a directory where you want to store the state for PMM.

3. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)

### Migrate from data container to host directory/volume

To migrate your PMM from data container to host directory or volume run the following command:

```sh
docker cp <containerId>:/srv /target/host/directory
```
