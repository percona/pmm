# API

## Interactive API documentation 
To integrate your applications or CI/CD, you can use our [online interactive documentation](https://percona-pmm.readme.io/reference/introduction).


## Swagger API documentation


PMM Server lets you visually interact with API resources representing all objects within PMM. You can browse the API using the [Swagger](https://swagger.io/tools/swagger-ui/) UI, accessible at the `/swagger` endpoint URL:

![!image](../images/PMM_Swagger_API_Get_Logs_View.jpg)

Clicking on an object lets you examine objects and execute requests on them:

![!image](../images/PMM_Swagger_API_Get_Logs_Execute.jpg)

The objects visible are nodes, services, and agents:

- A **Node** can be a bare metal server, a virtual machine, a Docker container, or a more specific type such as an Amazon RDS Node. A node runs zero or more Services and Agents, and has zero or more Agents providing insights for it.

- A **Service** is generally a database running on the Node: Amazon Aurora MySQL, MySQL, MongoDB, PostgreSQL, etc. It runs on zero (Amazon Aurora Serverless), single (MySQL), or several (Percona XtraDB Cluster) Nodes. It also has zero or more Agents providing insights for it.

- An **Agent** is an executable binary, that runs on the Node. It is not useful by itself, but instead provides insights (metrics, query performance data, etc.) about Nodes and/or Services. An agent always runs on the single Node (except External Exporters), and provides insights for zero or more Services and Nodes.

Nodes, Services, and Agents have **Types** which define their specific properties, and their specific logic.

Nodes and Services are external by nature – we do not manage them (create, destroy), but merely maintain their inventory (add to inventory, remove from inventory) in `pmm-managed`. Most Agents are started and stopped by `pmm-agent`. One exception is the External Exporter Type which is started externally.


### API Keys and authentication

API keys are used to control access to the PMM server components and resources. With an API key, you are authenticated to the PMM server, have access to PMM server components and resources, and perform various actions on them. You can use API keys as a replacement for basic authentication.

#### Generate API keys

PMM uses the Grafana API keys for authentication. Following are the steps to generate the API keys:

1. Login to PMM.
2. From the side menu, click *Configuration  → API keys*.
3. On the Configuration page, click *Add API Key*.
4. *Add API key* dialog box opens.
5. Enter  the following to generate an API key:
    - key name (you can give any desired name)
    - Select the Role from the drop-down 
    - Enter a value in the Time to live text box (hover on the tooltip for more information). 
    - Click Add.

6. *API Key Created* window displays your newly created key. Make sure to copy your key and keep it secure. 

#### Authenticate

You can authenticate your request using the HTTPS header.

!!! caution alert alert-warning "Important"
    Use the `-k` or `--insecure` parameter to force cURL to ignore invalid and self-signed SSL certificate errors. The option will skip the SSL verification process, and you can bypass any SSL errors while still having SSL-encrypted communication. However, using the `--insecure`  parameter is not recommended. Although the data transfer is encrypted, it is not entirely secure. For enhanced security of your PMM installation, you need valid SSL certificates. For information on validating SSL certificates, refer to: [SSL certificates](../how-to/secure.md).

```sh
curl -H "Authorization: Bearer <api_key>" https://127.0.0.1/v1/version
```

#### Use an API key in basic auth

You can use the API key for basic authentication in a REST API call in the following format. Replace `API_KEY` with your API key.

**Example**


```sh
curl -X GET https://api_key:API_KEY@127.0.0.1/v1/version
```
