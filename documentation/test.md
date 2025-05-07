# Set up PMM Client

To make the most of Percona Monitoring and Management (PMM), we suggest to set up a **PMM Client** in your database environment. This client acts as an agent, relaying back the key performance insights from your database to the PMM Server.

Depending on your setup, PMM Client may already be set up requiring only authentication or you may need to install it entirely. To understand what needs to be done in your system, select the database technology you're using first.

## Select your database technology and host

Select the database technology you're using so we can direct you to the best set up of your PMM Client and start collecting data signals back to PMM Server.


``` mermaid
graph RL
PS(PMM Server)
subgraph s2[Now: Step 2]
    DB[(Database)] -- Data collection --> PC(PMM Client)
end
PC -- Transmission --> PS
```

=== ":material-dolphin: MySQL"

    For MySQL check the type of host that you have for your database and follow the instructions required to set up PMM Client.

    | <small>*Host*</small> | <small>*Recommended set up*</small> | <small>*Other advanced options*</small> |
    | --------------------- | ----------------------------------- | ------------------------------ |
    | **Self-hosted / AWS EC2** | [**Install PMM Client using Percona Repositories** :material-arrow-right:](./install-pmm-client/percona-repositories.md) | [Using a PMM Client Docker image](#)<br><br>[Download and install PMM Client files](#) |
    | **AWS RDS / AWS Aurora** | [**Configure AWS settings** :material-arrow-right:](#) |
    | **Azure Database for MySQL** | [**Configure Azure Settings** :material-arrow-right:](#) |
    | **Google Cloud SQL for MySQL** | [**Configure Google Cloud Settings** :material-arrow-right:](#) |
    | **Other hosts / No access to the node** | [**Remote monitoring** :material-arrow-right:](#) |

=== ":material-elephant: PostgreSQL"

    PostgreSQL links here...

=== ":material-leaf: MongoDB"

    MongoDB links here...

=== ":material-database: ProxySQL"

    ProxySQL links here...

=== ":material-database: HAproxy"

    HAproxy links here...

=== ":simple-linux: Linux"

    Directly in Linux links here...

=== "Other technologies"

    Others' links here...