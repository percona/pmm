# Plugin issues

## PMM cannot install, upgrade, or remove plugins

If you cannot install, upgrade, or remove plugins in PMM, check the permissions on the `/srv/grafana/plugins` directory. 

When permissions are wrong, Grafana cannot write to the directory, which blocks all plugin operations.

## Solution

To fix the permissions and restart Grafana, run the following commands:
{.power-number}

1. Set ownership on the `/srv/grafana/plugins` directory to `1000:0`, which is the UID/GID that PMM Server runs as:

    ```sh
    docker exec --user root pmm-server chown -R 1000:0 /srv/grafana/plugins
    ```

2. Restart Grafana to pick up the change:

    ```sh
    docker exec --user root pmm-server supervisorctl restart grafana
    ```