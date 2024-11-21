# Connect ProxySQL instance

Use the `proxysql` alias to enable ProxySQL performance metrics monitoring.

## USAGE

```sh
pmm-admin add proxysql --username=pmm --password=pmm
```

where `username` and `password` are credentials for the administration interface of the monitored ProxySQL instance. 
You should configure a read-only account for monitoring using the [`admin-stats_credentials`](https://proxysql.com/documentation/global-variables/admin-variables/#admin-stats_credentials) variable in ProxySQL

Additionally, two positional arguments can be appended to the command line flags: a service name to be used by PMM, and a service address. If not specified, they are substituted automatically as `<node>-proxysql` and `127.0.0.1:6032`.

The output of this command may look as follows:

```sh
pmm-admin add proxysql --username=pmm --password=pmm
```

```text
ProxySQL Service added.
Service ID  : f69df379-6584-4db5-a896-f35ae8c97573
Service name: ubuntu-proxysql
```

Beside positional arguments shown above you can specify service name and
service address with the following flags: `--service-name`, and `--host` (the
hostname or IP address of the service) and `--port` (the port number of the
service), or `--socket` (the UNIX socket path). If both flag and positional argument are present, flag gains higher
priority. Here is the previous example modified to use these flags for both host/port or socket connections:

```sh
pmm-admin add proxysql --username=pmm --password=pmm --service-name=my-new-proxysql --host=127.0.0.1 --port=6032
pmm-admin add proxysql --username=pmm --password=pmm --service-name=my-new-proxysql --socket=/tmp/proxysql_admin.sock
```
