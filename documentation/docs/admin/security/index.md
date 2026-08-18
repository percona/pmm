# About security in PMM


By default, PMM ships with a self-signed certificate to enable usage out of the box. While this does enable users to have encrypted connections between clients (database clients and web/API clients) and the PMM Server, it shouldn't be considered a properly secured connection.  

Taking the following precautions will ensure that you are truly secure:

- [SSL encryption with trusted certificates](../../admin/security/ssl_encryption.md) to secure traffic between clients and server;

- [Grafana HTTPS secure cookies](../../admin/security/grafana_cookies.md)
- [Encrypt the PMM Client configuration file](client_config_encryption.md) to protect stored credentials on client hosts

## Manually configure the PostgreSQL Grafana datasource

Starting with PMM 3.9.0, PMM no longer provisions a PostgreSQL Grafana datasource by default. If you need to query PMM's internal PostgreSQL database directly, you can add the datasource manually.

To prevent unauthorized data modification, configure the datasource with a database user that has **SELECT**-only permissions.

To add the PostgreSQL datasource:
{.power-number}

1. From the **Home** page, locate the Search icon (top right on the screen).
2. Type "plugins" and select **Administration > Plugins and data > Plugins**.
3. Find and open the **Postgres** plugin.
4. Select **Add new datasource**.
5. Configure the connection parameters and click **Save & test**.
6. If the test is successful, you can start using the datasource from the **Explore** page.

!!! note

    You might see this datasource labeled as unsupported. You can safely disregard that label. The PostgreSQL datasource works as expected.

