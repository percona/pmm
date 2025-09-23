# MySQL secure_file_priv configuration option empty check

## Description

The option **secure_file_priv** defaults to NULL, which essentially allows users with FILE privilege to create files at any location where MySQL server has Write permission. This is considered less secure and goes against Percona's Best Practices.

## Resolution

To provide a more secure installation, the scope of FILE privilege should be restricted using a secure default value for **--secure-file-priv**. 
Edit **my.cnf** to provide secure-file-priv configuration and provide a specific location for users with FILE privileges to create files. This is not a dynamic variable and will need an instance reboot.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
