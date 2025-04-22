# Enable access control

You can enable access control in PMM as follows:

- Docker
- User Interface

## Enabling access control in container orchestration systems

To enable access control in container orchestration systems, pass the environment variable `PMM_ENABLE_ACCESS_CONTROL` when starting the container.

```sh
docker run … -e PMM_ENABLE_ACCESS_CONTROL=1
```

For `docker compose`, add the environment variable to the `docker-compose.yml` file:

```
services:
  pmm-server:
    …
    environment:
      …
      PMM_ENABLE_ACCESS_CONTROL=1
```

## Enabling access control from the UI

To enable access control from the UI:

From the main menu, go to  **PMM Configuration > Settings > Advanced Settings > Access Control** and click the <i class="uil uil-toggle-off"></i> toggle.
