# MongoDB Exporter (mongodb_exporter)

The following options may be passed to the `mongodb:metrics` monitoring service as additional options. For more information about this exporter see its GitHub repository:
[https://github.com/percona/mongodb_exporter](https://github.com/percona/mongodb_exporter).

## Options

| Name                                     | Description                                                           |
| ---------------------------------------- | --------------------------------------------------------------------- |
| -collect.collection                      | Enable collection of Collection metrics                               |
| -collect.database                        | Enable collection of Database metrics                                 |
| -groups.enabled string                   | Comma-separated list of groups to use, (default “asserts,durability,background_flushing,connections,extra_info,global_lock, index_counters,network,op_counters,op_counters_repl,memory,locks,metrics”) |
| -mongodb.max-connections int             | Max number of pooled connections to the database. (default 1)         |
| -mongodb.tls                             | Enable tls connection with mongo server                               |
| -mongodb.tls-ca string                   | Path to PEM file that contains the CAs that are trusted for server connections. If provided: MongoDB servers connecting to should present a certificate signed by one of this CAs. If not provided: System default CAs are used. |
| -mongodb.tls-cert string                 | Path to PEM file that contains the certificate (and optionally also the decrypted private key in PEM format). This should include the whole certificate chain. If provided: The connection will be opened via TLS to the MongoDB server. |
| -mongodb.tls-disable-hostname-validation | Do hostname validation for server connection. |
| -mongodb.tls-private-key string          | Path to PEM file that contains the decrypted private key (if not contained in mongodb.tls-cert file). |
| -mongodb.uri string                      | MongoDB URI, format: [mongodb://][user:pass@]host1[:port1][,host2[:port2],…][/database][?options] (default “mongodb://localhost:27017”) |
| -version                                 | Print version information and exit. |
| -web.auth-file string                    | Path to YAML file with server_user, server_password options for http basic auth (overrides HTTP_AUTH env var). |
| -web.listen-address string               | Address to listen on for web interface and telemetry. (default “:9216”) |
| -web.metrics-path string                 | Path under which to expose metrics. (default “/metrics”) |
| -web.ssl-cert-file string                | Path to SSL certificate file. |
| -web.ssl-key-file string                 | Path to SSL key file. |
