# User not using SSL protocol to connect

## Description
When MySQL server is configured with SSL, users can still connect to the server using `--ssl-mode=disabled`. The connection in this case will not be encrypted and data will be subject to sniffing. It is recommended when using SSL, to enforce it for any users. 



## Resolution
To prevent user to connect on insecure protocol, you can act at instance level:
```
[mysqld]
require_secure_transport=ON
```
Or when creating the user:
`CREATE USER 'jeffrey'@'localhost' REQUIRE SSL;`
 