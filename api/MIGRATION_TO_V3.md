## Migrations of API endpoints to make them more RESTful

| Current (v2)                                    | Migrate to (v3)                                | Comments                        |
| ----------------------------------------------- | ---------------------------------------------- | ------------------------------- |

**ServerService**                                   **ServerService**
GET /logz.zip                                       GET /v1/server/logs.zip                          /logs.zip is now a redirect to /v1/server/logs.zip
GET /v1/version                                     GET /v1/server/version                           ✅ /v1/version is now a redirect to /v1/server/version
GET /v1/readyz                                      GET /v1/server/readyz                            ✅ /v1/readyz is now a redirect to /v1/server/readyz
POST /v1/AWSInstanceCheck                           GET /v1/server/AWSInstance                       ✅
POST /v1/leaderHealthCheck                          GET /v1/server/leaderHealthCheck                 ✅
POST /v1/settings/Change                            PUT /v1/server/settings                          ✅
POST /v1/settings/Get                               GET /v1/server/settings                          ✅
POST /v1/updates/Check                              GET /v1/server/updates
POST /v1/updates/Start                              POST /v1/server/updates:start                  
POST /v1/updates/Status                             GET /v1/server/updates/status                    pass "auth_token" via headers

**UserService**                                     **UserService**
GET /v1/user                                        GET /v1/users/me                                 needs no {id} in path
PUT /v1/user                                        PUT /v1/users/me                                 needs no {id} in path
POST /v1/user/list                                  GET /v1/users/users

**AgentsService**                                   **AgentsService**
POST /v1/inventory/Agents/Add                       POST /v1/inventory/agents
POST /v1/inventory/Agents/Change                    PUT /v1/inventory/agents/{id}
POST /v1/inventory/Agents/Get                       GET /v1/inventory/agents/{id}
POST /v1/inventory/Agents/List                      GET /v1/inventory/agents
POST /v1/inventory/Agents/Remove                    DELETE /v1/inventory/agents/{id}
POST /v1/inventory/Agents/GetLogs                   GET /v1/inventory/agents/{id}/logs            

**NodesService**                                   **NodesService**
POST /v1/inventory/Nodes/Add                        POST /v1/inventory/nodes
POST /v1/inventory/Nodes/Get                        GET /v1/inventory/nodes/{id}
POST /v1/inventory/Nodes/Delete                     DELETE /v1/inventory/nodes/{id}
POST /v1/inventory/Nodes/List                       GET /v1/inventory/nodes

**ServicesService**                                 **ServicesService**
POST /v1/inventory/Services/Add                     POST /v1/inventory/services
POST /v1/inventory/Services/Change                  PUT /v1/inventory/services/{id}
POST /v1/inventory/Servicse/Get                     GET /v1/inventory/services/{id}
POST /v1/inventory/Services/List                    GET /v1/inventory/services
POST /v1/inventory/Services/Remove                  DELETE /v1/inventory/services/{id}               pass ?force=true to remove service with agents
POST /v1/inventory/Services/ListTypes               GET /v1/inventory/services/types
POST /v1/inventory/Services/CustomLabels/Add        POST /v1/inventory/services/{id}/custom_labels   !!! remove and refactor in favor of PUT /v1/inventory/services/{id}
POST /v1/inventory/Services/CustomLabels/Remove     DELETE /v1/inventory/services/{id}/custom_labels !!! remove and refactor in favor of PUT /v1/inventory/services/{id}

**ManagementService**                               **ManagementService**
POST /v1/management/Annotations/Add                 POST /v1/management/annotations
POST /v1/management/Node/Register                   POST /v1/management/nodes
POST /v1/management/External/Add                    POST /v1/management/services                     pass a service type in body
POST /v1/management/HAProxy/Add                     POST /v1/management/services                     pass a service type in body
POST /v1/management/MongoDB/Add                     POST /v1/management/services                     pass a service type in body
POST /v1/management/MySQL/Add                       POST /v1/management/services                     pass a service type in body
POST /v1/management/PostgreSQL/Add                  POST /v1/management/services                     pass a service type in body
POST /v1/management/ProxySQL/Add                    POST /v1/management/services                     pass a service type in body
POST /v1/management/RDS/Add                         POST /v1/management/services                     pass a service type in body
POST /v1/management/RDS/Discover                    POST /v1/management/services:discoverRDS
POST /v1/management/Service/Remove                  DELETE /v1/management/services/{id}              ({service_id} or {service_name}) and optional {service_type}

**ActionsService**                                  **ActionService**
POST /v1/actions/Cancel                             POST /v1/actions:cancel
POST /v1/actions/Get                                GET /v1/actions/{id}
POST /v1/actions/StartMongoDBExplain                POST /v1/actions:startServiceAction              NOTE: several similar actions are merged into one
POST /v1/actions/StartMySQLExplain                  POST /v1/actions:startServiceAction
POST /v1/actions/StartMySQLExplainJSON              POST /v1/actions:startServiceAction
POST /v1/actions/StartMySQLExplainTraditionalJSON   POST /v1/actions:startServiceAction
POST /v1/actions/StartMySQLShowCreateTable          POST /v1/actions:startServiceAction
POST /v1/actions/StartMySQLShowIndex                POST /v1/actions:startServiceAction
POST /v1/actions/StartMySQLShowTableStatus          POST /v1/actions:startServiceAction
POST /v1/actions/StartPTMongoDBSummary              POST /v1/actions:startServiceAction
POST /v1/actions/StartPTMySQLSummary                POST /v1/actions:startServiceAction
POST /v1/actions/StartPTPgSummary                   POST /v1/actions:startServiceAction
POST /v1/actions/StartPostgreSQLShowCreateTable     POST /v1/actions:startServiceAction
POST /v1/actions/StartPostgreSQLShowIndex           POST /v1/actions:startServiceAction
POST /v1/actions/StartPTSummary                     POST /v1/actions:startNodeAction

**AlertingService**                                 **AlertingService**
POST /v1/alerting/Rules/Create                      POST /v1/alerting/rules
POST /v1/alerting/Templates/Create                  POST /v1/alerting/templates
POST /v1/alerting/Templates/Update                  PUT /v1/alerting/templates/{name}            !!! pass yaml in body
POST /v1/alerting/Templates/List                    GET /v1/alerting/templates
POST /v1/alerting/Templates/Delete                  DELETE /v1/alerting/templates/{name}

**AdvisorService**                                 **AdvisorService**
POST /v1/advisors/Change                            POST /v1/advisors/checks:batchChange         !!! exception: updates multiple checks
<!-- POST /v1/advisors/FailedChecks                 POST /v1/advisors/checks:failedChecks        !!! try to implement as a GET request, see below -->
POST /v1/advisors/FailedChecks                      GET /v1/advisors/checks/failedChecks         ex: ?service_id=/service_id/1234-5678-abcd-efgh&page_params.page_size=100&page_params.index=1
POST /v1/advisors/List                              GET /v1/advisors
POST /v1/advisors/ListChecks                        GET /v1/advisors/checks
POST /v1/advisors/StartChecks                       POST /v1/advisors/checks:start
POST /v1/advisors/ListFailedServices                GET /v1/advisors/failedServices

**ArtifactsService**                                **ArtifactsService**                             TODO: merge to BackupService
POST /v1/backup/Artifacts/List                      GET /v1/backups/artifacts
POST /v1/backup/Artifacts/Delete                    DELETE /v1/backups/artifacts/{id}                ?remove_files=true
POST /v1/backup/Artifacts/PITRTimeranges            GET /v1/backups/artifacts/{id}/pitr_timeranges

**BackupsService**                                  **BackupService**                                TODO: rename to singular
POST /v1/backup/Backups/ChangeScheduled             PUT /v1/backups:changeScheduled
POST /v1/backup/Backups/GetLogs                     GET /v1/backups/{id}/logs
POST /v1/backup/Backups/ListArtifactCompatibleServices GET /v1/backups/{id}/services                 Could also be /compatible_services
POST /v1/backup/Backups/ListScheduled               GET /v1/backups/scheduled
POST /v1/backup/Backups/RemoveScheduled             GET /v1/backups/scheduled/{id}
<!-- POST /v1/backup/Backups/Restore                                                                 Moved to RestoreService -->
POST /v1/backup/Backups/Schedule                    POST /v1/backups:schedule
POST /v1/backup/Backups/Start                       POST /v1/backups:start

**LocationsService**                                **LocationsService**
POST /v1/backup/Locations/Add                       POST /v1/backups/locations
POST /v1/backup/Locations/Change                    PUT /v1/backups/locations/{id}                   Extract the location_id from the body to {id}
POST /v1/backup/Locations/List                      GET /v1/backups/locations
POST /v1/backup/Locations/Remove                    DELETE /v1/backups/locations/{id}                ?force=true
POST /v1/backup/Locations/TestConfig                POST /v1/backups/locations:testConfig

**RestoreHistoryService**                           **RestoreService**
POST /v1/backup/RestoreHistory/List                 GET /v1/backups/restores                         Note: could also be restore_history
POST /v1/backup/Backups/Restore                     POST /v1/backups/restores:start

**DumpsService**                                    **DumpService**                                  TODO: rename to singular
POST /v1/dump/List                                  GET /v1/dumps
POST /v1/dump/Delete                                POST /v1/dumps:batchDelete                       accepts an array in body
POST /v1/dump/GetLogs                               GET /v1/dumps/{id}/logs                          ?offset=10,limit=100
POST /v1/dump/Start                                 POST /v1/dumps:start                          
POST /v1/dump/Upload                                POST /v1/dumps:upload

**RoleService**                                     **AccessControlService**                         TODO: rename to AccessControlService
POST /v1/role/Assign                                POST /v1/accesscontrol/roles:assign
POST /v1/role/Create                                POST /v1/accesscontrol/roles
POST /v1/role/Delete                                DELETE /v1/accesscontrol/roles/{id}              ?replacement_role_id=id
POST /v1/role/Get                                   GET /v1/accesscontrol/roles/{id}
POST /v1/role/List                                  GET /v1/accesscontrol/roles
POST /v1/role/SetDefault                            POST /v1/accesscontrol/roles:setDefault
POST /v1/role/Update                                PUT /v1/accesscontrol/roles/{id}                 Extract the role_id from the body to {id}

**MgmtService**                                     **ManagementV1Beta1Service**                     NOTE: promoted to v1 from v1beta1
POST /v1/management/Agent/List                      GET /v1/management/agents
POST /v1/management/Node/Get                        GET /v1/management/nodes/{id}
POST /v1/management/Node/List                       GET /v1/management/nodes
POST /v1/management/AzureDatabase/Add               POST /v1/management/services/azure
POST /v1/management/AzureDatabase/Discover          POST /v1/management/services/azure:discover
POST /v1/management/Service/List                    GET /v1/management/services

**QANService**                                      **QANService**
POST /v1/qan/Filters/Get                            POST /v1/qan/metrics:getFilters                  accepts a bunch of params, incl. an array
POST /v1/qan/GetMetricsNames                        POST /v1/qan/metrics:getNames                    Note: it accepts no params, but hard to make it a GET
POST /v1/qan/GetReport                              POST /v1/qan/metrics:getReport
POST /v1/qan/ObjectDetails/ExplainFingerprintByQueryId POST /v1/qan:explainFingerprint
POST /v1/qan/ObjectDetails/GetHistogram             POST /v1/qan:getHistogram
POST /v1/qan/ObjectDetails/GetLables                POST /v1/qan:getLabels
POST /v1/qan/ObjectDetails/GetMetrics               POST /v1/qan:getMetrics
POST /v1/qan/ObjectDetails/GetQueryExample          POST /v1/qan/query:getExample                   !!! Need to revisit the endpoint design
POST /v1/qan/ObjectDetails/GetQueryPlan             GET /v1/qan/query/{query_id}/plan
POST /v1/qan/ObjectDetails/QueryExists              GET /v1/qan/query/{query_id}                    !!! Return query_id, fingerptint
POST /v1/qan/ObjectDetails/SchemaByQueryId          POST /v1/qan/query:getSchema

**PlatformService**                                 **PlatformService**
POST /v1/platform/Connect                           POST /v1/platform:connect
POST /v1/platform/Disconnect                        POST /v1/platform:disconnect
POST /v1/platform/GetContactInformation             GET /v1/platform/contact
POST /v1/platform/SearchOganizationEntitlemenets    GET /v1/platform/organization/entitlements
POST /v1/platform/SearchOganizationTickets          GET /v1/platform/organization/tickets
POST /v1/platform/ServerInfo                        GET /v1/platform/server
POST /v1/platform/UserInfo                          GET /v1/platform/user

// TODO: rename `period_start_from` to `start_from` and `period_start_to` to `start_to`


## The use of custom methods in RESTful API

We have a few custom methods in our RESTful API.

Custom methods refer to API methods besides the 5 standard methods. They should only be used for functionality that cannot be easily expressed via standard methods. In general, API designers should choose standard methods over custom methods whenever feasible. Standard Methods have simpler and well-defined semantics that most developers are familiar with, so they are easier to use and less error prone. Another advantage of standard methods is the API platform has better understanding and support for standard methods, such as billing, error handling, logging, monitoring.

A custom method can be associated with a resource, a collection, or a service. It may take an arbitrary request and return an arbitrary response, and also supports streaming request and response.

A custom method is always a POST request. The URL path must end with a suffix consisting of a colon followed by the custom verb, like in the following example:

```
https://service.name/v1/some/resource/name:customVerb
```

The custom method should be used in the following cases:

1. When the action cannot be performed by the standard RESTful methods. 
2. When the action performed is not idempotent.
3. When the action performed manipulates data, but does not fit into the standard CRUD operations.
4. When the action performed might contain sensitive data, that cannot be passed via URL query params.
