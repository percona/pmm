# Remove services from monitoring

To stop monitoring a service, use the `pmm-admin remove` command with the appropriate service type and name:

```sh
pmm-admin remove <service-type> <service-name>
```
## Command reference
- `service-type`: The type of service to remove: mysql, mongodb, postgresql, proxysql, haproxy, or external
- `service-name`: The name of the service as displayed in PMM inventory

## Example
To remove a MySQL service:
```sh
pmm-admin remove mysql mysql-prod-db1
```

## Verify service removal
After removing a service, you can verify it's no longer being monitored by listing all monitored services:

```sh
pmm-admin list
```

## Related topics
- [Percona release](https://www.percona.com/doc/percona-repo-config/percona-release.html)
- [PMM Client architecture](../../../../reference/index.md#pmm-client)
