# Grafana HTTPS secure cookies

To enable:
{.power-number}

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
