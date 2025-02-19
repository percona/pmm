# Configure access control

You can configure access control in PMM as follows:

- Docker
- User Interface

## Configure access control using Docker

To configure access roles in a ``pmm-server`` docker container, pass an additional environment variable ``ENABLE_RBAC=1`` when starting the container.

```sh
docker run … -e ENABLE_RBAC=1
```

For compose add an additional variable:

```
services:
  pmm-server:
    …
    environment:
      …
      ENABLE_RBAC=1
```

## Configure access control from the UI

To configure access control from the UI:

From the main menu, go to  **PMM Configuration > Settings > Advanced Settings > Access Control** and click <i class="uil uil-toggle-off"></i> toggle.