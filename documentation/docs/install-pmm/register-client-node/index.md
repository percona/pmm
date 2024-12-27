# Register client nodes on PMM Server

Register your client nodes with PMM Server.

```sh
pmm-admin config --server-insecure-tls --server-url=https://admin:admin@X.X.X.X:443
```

- `X.X.X.X` is the address of your PMM Server.
- `443` is the default port number.
- `admin`/`admin` is the default PMM username and password. This is the same account you use to log into the PMM user interface, which you had the option to change when first logging in.

!!! caution alert alert-warning "Important"
    Clients *must* be registered with the PMM Server using a secure channel. If you use http as your server URL, PMM will try to connect via https on port 443. If a TLS connection can't be established you will get an error and you must use https along with the appropriate secure port.

??? info "Example"

    Register on PMM Server with IP address `192.168.33.14` using the default `admin/admin` username and password, a node with IP address `192.168.33.23`, type `generic`, and name `mynode`.

    ```sh
    pmm-admin config --server-insecure-tls --server-url=https://admin:admin@192.168.33.14:443 192.168.33.23 generic mynode
    ```