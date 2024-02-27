## Migrations of API endpoints to make them more RESTful

| Current (v2)                                    | Migrate to (v3)                              | Comments                        |
| ----------------------------------------------- | -------------------------------------------- | ------------------------------- |

**ServerService**                                   **ServerService**
GET /logz.zip                                       GET /api/server/v1/logs.zip                    redirect to /logs.zip in swagger                                             
GET /v1/version                                     GET /api/server/v1/version                     redirect to /v1/version in swagger
POST /v1/readyz                                     GET /api/server/v1/readyz                                                           
POST /v1/AWSInstanceCheck                           GET /api/server/v1/AWSInstance                                                      
POST /v1/leaderHealthCheck                          GET /api/server/v1/leaderHealthCheck                                                
POST /v1/settings/Change                            PUT /api/server/v1/settings
POST /v1/settings/Get                               GET /api/server/v1/settings
POST /v1/updates/Check                              GET /api/server/v1/updates
POST /v1/updates/Start                              POST /api/server/v1/updates:start              !!!
POST /v1/updates/Status                             GET /api/server/v1/updates/status              pass "auth_token" via headers, ?log_offset=200

**UserService**                                     **UserService**
GET /v1/user                                        GET /api/users/v1/me                           needs no {id} in path
PUT /v1/user                                        PUT /api/users/v1/me                           needs no {id} in path
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
POST /v1/inventory/Services/Remove                  DELETE /api/inventory/v1/services/{id}            pass ?force=true to remove service with agents
POST /v1/inventory/Services/ListTypes               GET /api/inventory/v1/services/types
POST /v1/inventory/Services/CustomLabels/Add        POST /api/inventory/v1/services/{id}/custom_labels !!! remove and refactore in favor of PUT /api/inventory/v1/services/{id}
POST /v1/inventory/Services/CustomLabels/Remove     DELETE /api/inventory/v1/services/{id}/custom_labels !!! remove and refactore in favor of PUT /api/inventory/v1/services/{id}

**ManagementService**                               **ManagementService**
POST /v1/management/Annotations/Add                 POST /api/management/v1/annotations
POST /v1/management/Node/Register                   POST /api/management/v1/nodes
POST /v1/management/External/Add                    POST /api/management/v1/services                  pass a service type in body
POST /v1/management/HAProxy/Add                     POST /api/management/v1/services                  pass a service type in body
POST /v1/management/MongoDB/Add                     POST /api/management/v1/services                  pass a service type in body
POST /v1/management/MySQL/Add                       POST /api/management/v1/services                  pass a service type in body
POST /v1/management/PostgreSQL/Add                  POST /api/management/v1/services                  pass a service type in body
POST /v1/management/ProxySQL/Add                    POST /api/management/v1/services                  pass a service type in body
POST /v1/management/RDS/Add                         POST /api/management/v1/services                  pass a service type in body
POST /v1/management/RDS/Discover                    POST /api/management/v1/services:discoverRDS
POST /v1/management/Service/Remove                  DELETE /api/management/v1/services/{id}           ({service_id} or {service_name}) and optional {service_type}
<!-- POST /v1/management/Service/Remove                  DELETE /api/management/v1/services/{id}           {service_id_or_name} and optional {service_type} -->

**ActionsService**                                  **ActionService**
POST /v1/actions/Cancel                             POST /api/actions/v1/actions:cancel
POST /v1/actions/Get                                GET /api/actions/v1/actions/{id}

POST /v1/actions/StartMongoDBExplain                POST /api/actions/v1/actions:startServiceAction 
POST /v1/actions/StartPTSummary                     POST /api/actions/v1/actions:startNodeAction

POST /v1/actions/StartMongoDBExplain                POST /api/actions/v1/actions:startMongoDBExplain
POST /v1/actions/StartMySQLExplain                  POST /api/actions/v1/actions:startMySQLExplain
POST /v1/actions/StartMySQLExplainJSON              POST /api/actions/v1/actions:startMySQLExplainJSON
POST /v1/actions/StartMySQLExplainTraditionalJSON   POST /api/actions/v1/actions:startMySQLExplainTraditionalJSON
POST /v1/actions/StartMySQLShowCreateTable          POST /api/actions/v1/actions:startMySQLShowCreateTable
POST /v1/actions/StartMySQLShowIndex                POST /api/actions/v1/actions:startMySQLShowIndex
POST /v1/actions/StartMySQLShowTableStatus          POST /api/actions/v1/actions:startMySQLShowTableStatus
// NODE
POST /v1/actions/StartPTMongoDBSummary              POST /api/actions/v1/actions:startPTMongoDBSummary
POST /v1/actions/StartPTMySQLSummary                POST /api/actions/v1/actions:startPTMySQLSummary
POST /v1/actions/StartPTPgSummary                   POST /api/actions/v1/actions:startPTPgSummary
POST /v1/actions/StartPTSummary                     POST /api/actions/v1/actions:startPTSummary
POST /v1/actions/StartPostgreSQLShowCreateTable     POST /api/actions/v1/actions:startPostgreSQLShowCreateTable
POST /v1/actions/StartPostgreSQLShowIndex           POST /api/actions/v1/actions:startPostgreSQLShowIndex

**AlertingService**                                 **AlertingService**
POST /v1/alerting/Rules/Create                      POST /api/alerting/v1/rules
POST /v1/alerting/Templates/Create                  POST /api/alerting/v1/templates
POST /v1/alerting/Templates/Update                  PUT /api/alerting/v1/templates/{name}            !!! pass yaml in body
POST /v1/alerting/Templates/List                    GET /api/alerting/v1/templates
POST /v1/alerting/Templates/Delete                  DELETE /api/alerting/v1/templates/{name}

**AdvisorService**                                 **AdvisorService**
POST /v1/advisors/Change                            POST /api/advisors/v1/checks:change              !!! exception: updates multiple checks
POST /v1/advisors/FailedChecks                      POST /api/advisors/v1/checks:failedChecks        !!! exception: accepts a bunch of params
POST /v1/advisors/List                              GET /api/advisors/v1
POST /v1/advisors/ListChecks                        GET /api/advisors/v1/checks
POST /v1/advisors/StartChecks                       POST /api/advisors/v1/checks:start
POST /v1/advisors/ListFailedServices                GET /api/advisors/v1/failedServices

**ArtifactsService**                                **ArtifactsService**                              TODO: merge to BackupService
POST /v1/backup/Artifacts/List                      GET /api/backups/v1/artifacts
POST /v1/backup/Artifacts/Delete                    DELETE /api/backups/v1/artifacts/{id}             ?remove_files=true
POST /v1/backup/Artifacts/PITRTimeranges            GET /api/backups/v1/artifacts/{id}/pitr_timeranges

**BackupsService**                                  **BackupService**                                 TODO: rename to singular
POST /v1/backup/Backups/ChangeScheduled             PUT /api/backups/v1/backups:changeScheduled
POST /v1/backup/Backups/GetLogs                     GET /api/backups/v1/backups/{id}/logs
POST /v1/backup/Backups/ListArtifactCompatibleServices GET /api/backups/v1/backups/{id}/services      Could also be /compatible_services
POST /v1/backup/Backups/ListScheduled               GET /api/backups/v1/backups/scheduled
POST /v1/backup/Backups/RemoveScheduled             GET /api/backups/v1/backups/scheduled/{id}
POST /v1/backup/Backups/Restore                     POST /api/backups/v1/backups:restore
POST /v1/backup/Backups/Schedule                    POST /api/backups/v1/backups:schedule
POST /v1/backup/Backups/Start                       POST /api/backups/v1/backups:start

**LocationsService**                                **LocationsService**                              TODO: merge to BackupService
POST /v1/backup/Locations/Add                       POST /api/backups/v1/locations
POST /v1/backup/Locations/Change                    PUT /api/backups/v1/locations
POST /v1/backup/Locations/List                      GET /api/backups/v1/locations
POST /v1/backup/Locations/Remove                    DELETE /api/backups/v1/locations/{id}             ?force=true
POST /v1/backup/Locations/TestConfig                POST /api/backups/v1/locations:testConfig

**RestoreHistoryService**                           **RestoreHistoryService**                         TODO: merge to BackupService
POST /v1/backup/RestoreHistory/List                 GET /api/backups/v1/history                       Note: could also be restore_history

**DumpsService**                                    **DumpService**                                   TODO: rename to singular
POST /v1/dump/List                                  GET /api/dumps/v1/dumps
POST /v1/dump/Delete                                POST /api/dumps/v1/dumps:delete                   !!! exception: accepts an array in params, i.e. dump_ids=[id1,id2]
POST /v1/dump/GetLogs                               GET /api/dumps/v1/dumps/{id}/logs                 ?offset=10,limit=100
POST /v1/dump/Start                                 POST /api/dumps/v1/dumps:start                          
POST /v1/dump/Upload                                POST /api/dumps/v1/dumps:upload

**RoleService**                                     **AccessControlService**                          TODO: rename to AccessControlService
POST /v1/role/Assign                                POST /api/accesscontrol/v1/roles:assign
POST /v1/role/Create                                POST /api/accesscontrol/v1/roles
POST /v1/role/Delete                                DELETE /api/accesscontrol/v1/roles/{id}           ?replacement_role_id=id
POST /v1/role/Get                                   GET /api/accesscontrol/v1/roles/{id}
POST /v1/role/List                                  GET /api/accesscontrol/v1/roles
POST /v1/role/SetDefault                            POST /api/accesscontrol/v1/roles:setDefault
POST /v1/role/Update                                POST /api/accesscontrol/v1/roles:update

**MgmtService**                                     **MgmtService**                                   Q: what should be the name?
POST /v1/management/Agent/List                      GET /api/management/v1/agents
POST /v1/management/Node/Get                        GET /api/management/v1/nodes/{id}
POST /v1/management/Node/List                       GET /api/management/v1/nodes
POST /v1/management/AzureDatabase/Add               POST /api/management/v1/services/azure
POST /v1/management/AzureDatabase/Discover          POST /api/management/v1/services/azure:discover
POST /v1/management/Service/List                    GET /api/management/v1/services

**QANService**                                      **QANService**
POST /v1/qan/Filters/Get                            POST /api/qan/v1/filters:get                      !!! exception: accepts a bunch of params, incl. an array
POST /v1/qan/GetMetricsNames                        POST /api/qan/v1/metrics:names                    Note: it accepts no params, but hard to make it a GET
POST /v1/qan/GetReport                              POST /api/qan/v1/metrics:report
POST /v1/qan/ObjectDetails/ExmplainFingerprintByQueryId POST /api/qan/v1/data:explainFingerprint
POST /v1/qan/ObjectDetails/GetHistogram             POST /api/qan/v1/data:histogram
POST /v1/qan/ObjectDetails/GetLables                POST /api/qan/v1/data:labels
POST /v1/qan/ObjectDetails/GetMetrics               POST /api/qan/v1/data:metrics
POST /v1/qan/ObjectDetails/GetQueryExample          POST /api/qan/v1/data:queryExample
POST /v1/qan/ObjectDetails/GetQueryPlan             POST /api/qan/v1/data:queryPlan
POST /v1/qan/ObjectDetails/QueryExists              POST /api/qan/v1/data:queryExists
POST /v1/qan/ObjectDetails/SchemaByQueryId          POST /api/qan/v1/data:schema

**PlatformService**                                 **PlatformService**
POST /v1/platform/Connect                           POST /api/platform/v1/platform:connect
POST /v1/platform/Disconnect                        POST /api/platform/v1/platform:disconnect
POST /v1/platform/GetContactInformation             GET /api/platform/v1/contact
POST /v1/platform/SearchOganizationEntitlemenets    POST /api/platform/v1/organization:searchEntitlements   Note: it accepts no params, but hard to make it a GET
POST /v1/platform/SearchOganizationTickets          POST /api/platform/v1/organization:searchTickets        Note: it accepts no params, but hard to make it a GET
POST /v1/platform/ServerInfo                        GET /api/platform/v1/server
POST /v1/platform/UserInfo                          GET /api/platform/v1/user
