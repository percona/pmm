# Install PMM Server on a Docker image

!!! note
    Before running PMM Server on our Docker image make sure you have Docker 1.12.6 or higher.

You 3 options on how you can store data from your PMM Server. Select the method below that best fits your case.

=== "Docker Volume (recommended)"
    
    A Docker Volume is a designated directory on the host machine or on a remote server that preserves data, ensuring it remains intact even if the container is removed. Ideal for backing up, restoring, or handling data on remote hosts or cloud providers.

    Here's how to do it in the steps below.
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

=== "Data Container"

    Instructions here...

=== "Host Directory"

    Instructions here...

## Next step: Set up PMM Client

Now that you have PMM Server set up we need to go to your databases and add PMM Client so that PMM Server can communicate with your databases. Learn how in the button below.

[Set up PMM Client :material-arrow-right:](../set-up-pmm-client/index.md){ .md-button .md-button--primary }

If you want to try something else before anything, here are some other ideas for next steps:

- [Backup](#) PMM Server and its data
- [Update](#) PMM Server
- [Restore](#) PMM Server
- [Remove](#) PMM Server
- [Use Environment Variables](#) to set PMM Server parameters.