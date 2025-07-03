# Register client nodes on PMM Server

Register your nodes to be monitored by PMM Server using the PMM Client:

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

where: 

- `X.X.X.X` is the address of your PMM Server
- `443` is the default port number
- `admin`/`admin` is the default PMM username and password. This is the same account you use to log into the PMM user interface, which you had the option to change when first logging in.

!!! caution alert alert-warning "HTTPS connection required"
    Nodes *must* be registered with the PMM Server using a secure HTTPS connection. If you try to use HTTP in your server URL, PMM will automatically attempt to establish an HTTPS connection on port 443. If a TLS connection cannot be established, you will receive an error message and must explicitly use HTTPS with the appropriate secure port.

??? info "Registration example"

    Register a node with IP address 192.168.33.23, type generic, and name mynode on a PMM Server with IP address 192.168.33.14:

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
    ```

## Related topics

- [PMM Client overview](../install-pmm-client/index.md) 
- [Client installation prerequisites](../install-pmm-client/prerequisites.md) 
- [Install PMM Client with Docker](../install-pmm-client/docker.md) 
- [Connect MySQL database](../install-pmm-client/connect-database/mysql/mysql.md)
- [Connect PostgreSQL database](../install-pmm-client/connect-database/postgresql.md) 
- [Connect MongoDB database](../install-pmm-client/connect-database/mongodb.md)
- [Add Linux system monitoring](../install-pmm-client/connect-database/linux.md) 
- [Unregister PMM Client](../../uninstall-pmm/unregister_client.md)