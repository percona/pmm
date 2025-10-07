# Advisor checks: User not using SSL protocol to connect

## Description

When MySQL server is configured with SSL, users can still connect to the server using `--ssl-mode=disabled`.

In this case, the connection will not be encrypted and data will be subject to sniffing.

When using SSL, it is recommended to enforce it for all users.

## Resolution

To prevent users from connecting using an insecure protocol, you can act at instance-level:

```text
[mysqld]
require_secure_transport=ON
```

Or when creating the user:

`CREATE USER 'jeffrey'@'localhost' REQUIRE SSL;`
 
## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
