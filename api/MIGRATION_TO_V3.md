
```bash
// POST /actions/v1/action:start
	body = {
		MongoDBExplain: {
			action_name: MongoDBExplain
		}
	}
```

| Current (v2)                                    | Migrate to (v3)                              | Comments                        |
| ----------------------------------------------- | -------------------------------------------- | ------------------------------- |
**ServerService**                                   **ServerService**
GET /logz.zip                                       GET /api/server/v1/logs.zip                        redirect to /logs.zip in swagger                                             
GET /v1/version                                     GET /api/server/v1/version                         redirect to /v1/version in swagger
POST /v1/readyz                                     GET /api/server/v1/readyz                                                           
POST /v1/AWSInstanceCheck                           GET /api/server/v1/AWSInstance                                                      
POST /v1/leaderHealthCheck                          GET /api/server/v1/leaderHealthCheck                                                
POST /v1/settings/Change                            PUT /api/server/v1/settings
POST /v1/settings/Get                               GET /api/server/v1/settings
POST /v1/updates/Check                              GET /api/server/v1/updates
POST /v1/updates/Start                              POST /api/server/v1/updates:start                 !!!
POST /v1/updates/Status                             GET /api/server/v1/updates/status?log_offset=200  "auth_token" - pass via headers |

**UserService**                                     **UserService**
GET /v1/user                                        GET /api/users/v1/user                            Needs no {id} in path
PUT /v1/user                                        PUT /api/users/v1/user                            Needs no {id} in path
POST /v1/user/list                                  GET /api/users/v1

**AgentsService**                                   **AgentsService**
POST /v1/inventory/Agents/Add                       POST /api/inventory/v1/agents
POST /v1/inventory/Agents/Change                    PUT /api/inventory/v1/agents/{id}
POST /v1/inventory/Agents/Get                       GET /api/inventory/v1/agents/{id}
POST /v1/inventory/Agents/List                      GET /api/inventory/v1/agents
POST /v1/inventory/Agents/Remove                    DELETE /api/inventory/v1/agents/{id}
POST /v1/inventory/Agents/GetLogs                   GET /api/inventory/v1/agents/{id}/logs            

**NodesService**                                   **NodesService**
POST /v1/inventory/Nodes/Add                        POST /api/inventory/v1/nodes
POST /v1/inventory/Nodes/Get                        GET /api/inventory/v1/nodes/{id}
POST /v1/inventory/Nodes/Delete                     DELETE /api/inventory/v1/nodes/{id}
POST /v1/inventory/Nodes/List                       GET /api/inventory/v1/nodes

**ServicesService**                                 **ServicesService**
POST /v1/inventory/Services/Add                     POST /api/management/v1/services
POST /v1/inventory/Services/Change                  PUT /api/inventory/v1/services/{id}
POST /v1/inventory/Servicse/Get                     GET /api/inventory/v1/services/{id}
POST /v1/inventory/Services/List                    GET /api/inventory/v1/services
POST /v1/inventory/Services/Remove                  DELETE /api/inventory/v1/services/{id}
POST /v1/inventory/Services/ListTypes               GET /api/inventory/v1/services/types
POST /v1/inventory/Services/CustomLabels/Add        POST /api/inventory/v1/services/custom_labels
POST /v1/inventory/Services/CustomLabels/Remove     DELETE /api/inventory/v1/services/custom_labels/{id}




--------------------------------------------------------------------------------------------------------------------------------
POST /v1/management/HAProxy/Add                     POST /management/v1/services/HAProxy       
POST /v1/management/Service/Remove                  DELETE /management/v1/services/{id}            {service_id} or {service_name}
