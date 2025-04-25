# Enable access control

Access control in PMM lets you restrict user access to specific metrics based on their roles. 
Choose your preferred method to enable this feature:

=== "Via Docker"

    When deploying PMM Server with Docker, enable access control by passing an environment variable:
    
    ```sh
    docker run -d \
      --name pmm-server \
      -p 443:443 \
      -e PMM_ENABLE_ACCESS_CONTROL=1 \
      percona/pmm-server:latest
    ```

=== "Via Docker Compose"

    For Docker Compose deployments, add the environment variable to your `docker-compose.yml` file:
    
    ```yaml
    services:
      pmm-server:
        image: percona/pmm-server:latest
        ports:
          - "443:443"
        environment:
          - PMM_ENABLE_ACCESS_CONTROL=1
        volumes:
          - pmm-data:/srv
    ```

=== "Via user interface"

    To enable access control from the PMM web interface:
    {.power-number}
    
    1. Log in to PMM with an administrator account.
    2. From the main menu, go to **PMM Configuration > Settings > Advanced Settings > Access Control**.
    3. Toggle the <i class="uil uil-toggle-off"></i> toggle.
    4. Click **Apply changes** to save your settings.

## After enabling access control

Once access control is enabled:

- All existing users will have full access until you assign specific roles.
- [Create access roles](../access-control/create_roles.md) for different user types.
- [Assign the new roles](../index.md) to your PMM users.
- Test that restrictions work as expected.