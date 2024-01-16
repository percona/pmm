
# Run Docker with volume

To run Docker with volume:
{.power-number}

1. Pull the image.

    ```sh
    docker pull percona/pmm-server:2
    ```

2. Create a volume:

    ```sh
    docker volume create pmm-data
    ```

3. Run the image:

    ```sh
    docker run --detach --restart always \
    --publish 443:443 \
    -v pmm-data:/srv \
    --name pmm-server \
    percona/pmm-server:2
    ```
    
4. Change the password for the default `admin` user.

    * For PMM versions 2.27.0 and later:

        ```sh
        docker exec -t pmm-server change-admin-password <new_password>
        ```

    * For PMM versions prior to 2.27.0:

        ```sh
        docker exec -t pmm-server bash -c 'grafana-cli --homepath /usr/share/grafana --configOverrides cfg:default.paths.data=/srv/grafana admin reset-admin-password newpass'
        ```

5. Visit `https://localhost:443` to see the PMM user interface in a web browser. (If you are accessing the docker host remotely, replace `localhost` with the IP or server name of the host.)
       

