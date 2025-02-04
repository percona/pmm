## Migrations of v2 API endpoints to v3

| Current (v2)                                               | Migrate to (v3)                                  | Comments                                            |
| ---------------------------------------------------------- | ------------------------------------------------ | --------------------------------------------------- |

  **Server**
  GET /logz.zip                                                GET /v1/server/logs.zip                            ✅ /logs.zip is redirected to /v1/server/logs.zip
  GET /v1/version                                              GET /v1/server/version                             ✅ /v1/version is redirected to /v1/server/version
  GET /v1/readyz                                               GET /v1/server/readyz                              ✅ /v1/readyz is redirected to /v1/server/readyz
  POST /v1/AWSInstanceCheck                                    GET /v1/server/AWSInstance                         ✅
  POST /v1/leaderHealthCheck                                   GET /v1/server/leaderHealthCheck                   ✅
  POST /v1/settings/Change                                     PUT /v1/server/settings                            ✅
  POST /v1/settings/Get                                        GET /v1/server/settings                            ✅
  POST /v1/settings/TestEmailAlertingSettings                  N/A                                                ❌ Removed in v3
  POST /v1/updates/Check                                       GET /v1/server/updates                             ✅
  POST /v1/updates/Start                                       POST /v1/server/updates:start                      ✅
  POST /v1/updates/Status                                      POST /v1/server/updates:getStatus                  ✅ auth_token is passed in the body

  **User**
  GET /v1/user                                                 GET /v1/users/me                                   ✅
  PUT /v1/user                                                 PUT /v1/users/me                                   ✅
  POST /v1/user/list                                           GET /v1/users                                      ✅ 

  **Inventory:: Agents**
  POST /v1/inventory/Agents/AddAzureDatabaseExporter           POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddExternalExporter                POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddMongoDBExporter                 POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddMySQLdExporter                  POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddNodeExporter                    POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddPMMAgent                        POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddPostgresExporter                POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddProxySQLExporter                POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddQANMongoDBProfilerAgent         POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddQANMySQLPerfSchemaAgent         POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddQANMySQLSlowlogAgent            POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddQANPostgreSQLPgStatMonitorAgent POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddQANPostgreSQLPgStatMonitorAgent POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/AddRDSExporter                     POST /v1/inventory/agents                          ✅ 
  POST /v1/inventory/Agents/ChangeAzureDatabaseExporter        PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeExternalExporter             PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeMongoDBExporter              PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeMySQLdExporter               PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeNodeExporter                 PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangePostgresExporter             PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeProxySQLExporter             PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/ChangeQANMongoDBProfilerAgent      PUT /v1/inventory/agents/{agent_id}                ✅
  POST /v1/inventory/Agents/ChangeQANMySQLPerfSchemaAgent      PUT /v1/inventory/agents/{agent_id}                ✅
  POST /v1/inventory/Agents/ChangeQANMySQLSlowlogAgent         PUT /v1/inventory/agents/{agent_id}                ✅
  POST /v1/inventory/Agents/ChangeQANPostgreSQLPgStatMonitorAgent PUT /v1/inventory/agents/{agent_id}             ✅ 
  POST /v1/inventory/Agents/ChangeQANPostgreSQLPgStatMonitorAgent PUT /v1/inventory/agents/{agent_id}             ✅ 
  POST /v1/inventory/Agents/ChangeRDSExporter                  PUT /v1/inventory/agents/{agent_id}                ✅ 
  POST /v1/inventory/Agents/Get                                GET /v1/inventory/agents/{agent_id}                ✅
  POST /v1/inventory/Agents/GetLogs                            GET /v1/inventory/agents/{agent_id}/logs           ✅
  POST /v1/inventory/Agents/List                               GET /v1/inventory/agents                           ✅ Query param filters: service_id, node_id 
  POST /v1/inventory/Agents/Remove                             DELETE /v1/inventory/agents/{agent_id}             ✅

  **Inventory:: Nodes**
  POST /v1/inventory/Nodes/Add                                 POST /v1/inventory/nodes                           ✅
  POST /v1/inventory/Nodes/AddContainer                        see POST /v1/inventory/nodes                       ❌ Deprecated in v2 and removed in v3
  POST /v1/inventory/Nodes/AddGeneric                          see POST /v1/inventory/nodes                       ❌ Deprecated in v2 and removed in v3
  POST /v1/inventory/Nodes/AddRemote                           see POST /v1/inventory/nodes                       ❌ Deprecated in v2 and removed in v3
  POST /v1/inventory/Nodes/AddRemoteAzureDatabase              see POST /v1/inventory/nodes                       ❌ Deprecated in v2 and removed in v3
  POST /v1/inventory/Nodes/AddRemoteRDS                        see POST /v1/inventory/nodes                       ❌ Deprecated in v2 and removed in v3
  POST /v1/inventory/Nodes/Get                                 GET /v1/inventory/nodes/{node_id}                  ✅
  POST /v1/inventory/Nodes/List                                GET /v1/inventory/nodes                            ✅
  POST /v1/inventory/Nodes/Remove                              DELETE /v1/inventory/nodes/{node_id}               ✅

  **Inventory:: Services**
  POST /v1/inventory/Services/AddExternalService               POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/AddHAProxyService                POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/AddMongoDB                       POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/AddMySQL                         POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/AddPostgreSQL                    POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/AddProxySQL                      POST /v1/inventory/services                        ✅
  POST /v1/inventory/Services/Change                           PUT /v1/inventory/services/{service_id}            ✅
  POST /v1/inventory/Servicse/Get                              GET /v1/inventory/services/{service_id}            ✅
  POST /v1/inventory/Services/List                             GET /v1/inventory/services                         ✅
  POST /v1/inventory/Services/Remove                           DELETE /v1/inventory/services/{service_id}         ✅ pass ?force=true to remove a service with agents
  POST /v1/inventory/Services/ListTypes                        POST /v1/inventory/services:getTypes               ✅
  POST /v1/inventory/Services/CustomLabels/Add                 PUT /v1/inventory/services/{service_id}            ✅
  POST /v1/inventory/Services/CustomLabels/Remove              PUT /v1/inventory/services/{service_id}            ✅

  **Management:: Actions**
  POST /v1/management/actions/Cancel                           POST /v1/actions:cancelAction                      ✅
  POST /v1/management/actions/Get                              GET /v1/actions/{action_id}                        ✅
  POST /v1/management/actions/StartMySQLExplain                POST /v1/actions:startServiceAction                ✅ Several endpoints merged into one
  POST /v1/management/actions/StartMySQLExplainJSON            POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartMySQLExplainTraditionalJSON POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartMySQLShowIndex              POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartMySQLShowCreateTable        POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartMySQLShowTableStatus        POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPostgreSQLShowCreateTable   POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPostgreSQLShowIndex         POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartMongoDBExplain              POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPTMongoDBSummary            POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPTMySQLSummary              POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPTPgSummary                 POST /v1/actions:startServiceAction                ✅
  POST /v1/management/actions/StartPTSummary                   POST /v1/actions:startNodeAction                   ✅

  **Management**
  POST /v1/management/Annotations/Add                          POST /v1/management/annotations                    ✅
  POST /v1/management/Agent/List                               GET /v1/management/agents                          ✅
  POST /v1/management/Node/Register                            POST /v1/management/nodes                          ✅
  POST /v1/management/Node/Unregister                          DELETE /v1/management/nodes/{node_id}              ✅ ?force=true
  POST /v1/management/Node/Get                                 GET /v1/management/nodes/{node_id}                 ✅
  POST /v1/management/Node/List                                GET /v1/management/nodes                           ✅
  POST /v1/management/External/Add                             POST /v1/management/services                       ✅ 
  POST /v1/management/HAProxy/Add                              POST /v1/management/services                       ✅
  POST /v1/management/MongoDB/Add                              POST /v1/management/services                       ✅
  POST /v1/management/MySQL/Add                                POST /v1/management/services                       ✅
  POST /v1/management/PostgreSQL/Add                           POST /v1/management/services                       ✅
  POST /v1/management/ProxySQL/Add                             POST /v1/management/services                       ✅
  POST /v1/management/RDS/Add                                  POST /v1/management/services                       ✅
  POST /v1/management/RDS/Discover                             POST /v1/management/services:discoverRDS           ✅
  POST /v1/management/azure/AzureDatabase/Add                  POST /v1/management/services/azure                 ✅
  POST /v1/management/azure/AzureDatabase/Discover             POST /v1/management/services:discoverAzure         ✅
  POST /v1/management/Service/List                             GET /v1/management/services                        ✅
  POST /v1/management/Service/Remove                           DELETE /v1/management/services/{service_id}        ✅ In addition, it accepts ?service_type=  

  **Alerting**
  POST /v1/management/alerting/Rules/Create                    POST /v1/alerting/rules                            ✅
  POST /v1/management/alerting/Templates/Create                POST /v1/alerting/templates                        ✅
  POST /v1/management/alerting/Templates/Update                PUT /v1/alerting/templates/{name}                  ✅
  POST /v1/management/alerting/Templates/List                  GET /v1/alerting/templates                         ✅
  POST /v1/management/alerting/Templates/Delete                DELETE /v1/alerting/templates/{name}               ✅

  **Advisors**
  POST /v1/management/Advisors/List                            GET /v1/advisors                                   ✅
  POST /v1/management/SecurityChecks/Change                    POST /v1/advisors/checks:batchChange               ✅
  POST /v1/management/SecurityChecks/FailedChecks              GET /v1/advisors/checks/failed                     ✅ ?service_id=1234&page_size=100&page_index=1
  POST /v1/management/SecurityChecks/List                      GET /v1/advisors/checks                            ✅
  POST /v1/management/SecurityChecks/Start                     POST /v1/advisors/checks:start                     ✅
  POST /v1/management/SecurityChecks/ListFailedServices        GET /v1/advisors/failedServices                    ✅
  POST /v1/management/SecurityChecks/GetCheckResults           N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/SecurityChecks/ToggleCheckAlert          N/A                                                ❌ Deprecated in v2 and removed in v3

  **Backups**
  POST /v1/backup/Backups/ChangeScheduled                      PUT /v1/backups:changeScheduled                    ✅
  POST /v1/backup/Backups/GetLogs                              GET /v1/backups/{artifact_id}/logs                 ✅
  POST /v1/backup/Backups/ListArtifactCompatibleServices       GET /v1/backups/{artifact_id}/compatible-services  ✅
  POST /v1/backup/Backups/ListScheduled                        GET /v1/backups/scheduled                          ✅
  POST /v1/backup/Backups/RemoveScheduled                      DELETE /v1/backups/scheduled/{scheduled_backup_id} ✅
  POST /v1/backup/Backups/Schedule                             POST /v1/backups:schedule                          ✅
  POST /v1/backup/Backups/Start                                POST /v1/backups:start                             ✅
  POST /v1/backup/Artifacts/List                               GET /v1/backups/artifacts                          ✅
  POST /v1/backup/Artifacts/Delete                             DELETE /v1/backups/artifacts/{artifact_id}         ✅ ?remove_files=true
  POST /v1/backup/Artifacts/PITRTimeranges                     GET /v1/backups/artifacts/{artifact_id}/pitr-timeranges ✅

  **Backups:: Locations**
  POST /v1/backup/Locations/Add                                POST /v1/backups/locations                         ✅
  POST /v1/backup/Locations/Change                             PUT /v1/backups/locations/{location_id}            ✅
  POST /v1/backup/Locations/List                               GET /v1/backups/locations                          ✅
  POST /v1/backup/Locations/Remove                             DELETE /v1/backups/locations/{location_id}         ✅ ?force=true
  POST /v1/backup/Locations/TestConfig                         POST /v1/backups/locations:testConfig              ✅

  **Backups:: Restore**
  POST /v1/backup/RestoreHistory/List                          GET /v1/backups/restores                           ✅
  POST /v1/backup/Backups/Restore                              POST /v1/backups/restores:start                    ✅
                                                               GET /v1/backups/restores/{restore_id}/logs         🆕 new, similar to /v1/backups/{artifact_id}/logs

  **Dumps**
  POST /v1/management/dump/Dumps/List                          GET /v1/dumps                                      ✅
  POST /v1/management/dump/Dumps/Delete                        POST /v1/dumps:batchDelete                         ✅ accepts an array in body
  POST /v1/management/dump/Dumps/GetLogs                       GET /v1/dumps/{dump_id}/logs                       ✅ ?offset=0&limit=100
  POST /v1/management/dump/Dumps/Start                         POST /v1/dumps:start                               ✅              
  POST /v1/management/dump/Dumps/Upload                        POST /v1/dumps:upload                              ✅

  **AccessControl**
  POST /v1/management/Role/Assign                              POST /v1/accesscontrol/roles:assign                ✅
  POST /v1/management/Role/Create                              POST /v1/accesscontrol/roles                       ✅
  POST /v1/management/Role/Delete                              DELETE /v1/accesscontrol/roles/{role_id}           ✅ ?replacement_role_id=abcdedf-123456
  POST /v1/management/Role/Get                                 GET /v1/accesscontrol/roles/{role_id}              ✅
  POST /v1/management/Role/List                                GET /v1/accesscontrol/roles                        ✅
  POST /v1/management/Role/SetDefault                          POST /v1/accesscontrol/roles:setDefault            ✅
  POST /v1/management/Role/Update                              PUT /v1/accesscontrol/roles/{role_id}              ✅

  **Management:: Integrated Alerting**
  POST /v1/management/ia/Alerts/List                           N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Alerts/Toggle                         N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Channels/Add                          N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Channels/Change                       N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Channels/List                         N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Channels/Remove                       N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Rules/Create                          N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Rules/Delete                          N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Rules/List                            N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Rules/Toggle                          N/A                                                ❌ Deprecated in v2 and removed in v3
  POST /v1/management/ia/Rules/Update                          N/A                                                ❌ Deprecated in v2 and removed in v3

  **QAN**
  POST /v0/qan/Filters/Get                                     POST /v1/qan/metrics:getFilters                    ✅
  POST /v0/qan/GetMetricsNames                                 POST /v1/qan/metrics:getNames                      ✅
  POST /v0/qan/GetReport                                       POST /v1/qan/metrics:getReport                     ✅
  POST /v0/qan/ObjectDetails/ExplainFingerprintByQueryID       POST /v1/qan:explainFingerprint                    ✅
  POST /v0/qan/ObjectDetails/GetHistogram                      POST /v1/qan:getHistogram                          ✅
  POST /v0/qan/ObjectDetails/GetLables                         POST /v1/qan:getLabels                             ✅
  POST /v0/qan/ObjectDetails/GetMetrics                        POST /v1/qan:getMetrics                            ✅
  POST /v0/qan/ObjectDetails/GetQueryPlan                      GET /v1/qan/query/{queryid}/plan                   ✅
  POST /v0/qan/ObjectDetails/QueryExists                       POST /v1/qan/query:exists                          ✅ 
  POST /v0/qan/ObjectDetails/GetQueryExample                   POST /v1/qan/query:getExample                      ✅
  POST /v0/qan/ObjectDetails/SchemaByQueryID                   POST /v1/qan/query:getSchema                       ✅

  **Platform**
  POST /v1/Platform/Connect                                    POST /v1/platform:connect                          ✅
  POST /v1/Platform/Disconnect                                 POST /v1/platform:disconnect                       ✅
  POST /v1/Platform/GetContactInformation                      GET /v1/platform/contact                           ✅
  POST /v1/Platform/SearchOganizationEntitlemenets             GET /v1/platform/organization/entitlements         ✅
  POST /v1/Platform/SearchOganizationTickets                   GET /v1/platform/organization/tickets              ✅
  POST /v1/Platform/ServerInfo                                 GET /v1/platform/server                            ✅
  POST /v1/Platform/UserStatus                                 GET /v1/platform/user                              ✅

  // TODO: rename `period_start_from` to `start_from` and `period_start_to` to `start_to`
