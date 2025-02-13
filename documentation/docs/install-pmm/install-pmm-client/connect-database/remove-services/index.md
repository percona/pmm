# Remove services from monitoring

You must specify the service type and service name to remove services from monitoring.

```sh
pmm-admin remove <service-type> <service-name>
```

`service-type`
: One of `mysql`, `mongodb`, `postgresql`, `proxysql`, `haproxy`, `external`.

!!! seealso alert alert-info "See also"
    - [Percona release](https://www.percona.com/doc/percona-repo-config/percona-release.html)
    - [PMM Client architecture](../../../../reference/index.md#pmm-client1)
