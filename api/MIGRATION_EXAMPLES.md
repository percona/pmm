# API Migration Examples

## Some dos and don'ts

### Don't URLEncode the prefix - it won't work
curl -s -X PUT -d '{"postgres_exporter":{"enable":false}}' "http://admin:admin@127.0.0.1:8080/v1/inventory/agents/%2Fagent_id%2Ff56ee4e8-116c-496b-812f-a803dd2fe88d"

### Don't use plain bold prefix - it won't work
curl -s -X PUT -d '{"postgres_exporter":{"enable":false}}' "http://admin:admin@127.0.0.1:8080/v1/inventory/agents//agent_id/f56ee4e8-116c-496b-812f-a803dd2fe88d"

### Do pass UUID as an URL path segment
curl -s -X PUT -d '{"postgres_exporter":{"enable":false}}' http://admin:admin@127.0.0.1:8080/v1/inventory/agents/f56ee4e8-116c-496b-812f-a803dd2fe88d

## Examples

### POST /v1/inventory/Agents/Change -> PUT /v1/inventory/agents/{agent_id}
curl -s -X PUT -d '{"postgres_exporter":{"enable":true}}' http://admin:admin@127.0.0.1:8080/v1/inventory/agents/f56ee4e8-116c-496b-812f-a803dd2fe88d

### POST /v1/inventory/Agents/Get -> GET /v1/inventory/agents/{agent_id}
curl -s -X GET http://admin:admin@127.0.0.1:8080/v1/inventory/agents/02ecd9e3-d7b8-4d94-9c75-060b8e6e3e84

### POST /v1/inventory/Agents/List -> GET /v1/inventory/agents?agent_type=AGENT_TYPE_POSTGRES_EXPORTER
curl -s -X GET http://admin:admin@127.0.0.1:8080/v1/inventory/agents
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?agent_type=AGENT_TYPE_POSTGRES_EXPORTER
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?agent_type=AGENT_TYPE_PMM_AGENT
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?pmm_agent_id=pmm-server
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?pmm_agent_id=/agent_id/02ecd9e3-d7b8-4d94-9c75-060b8e6e3e84
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?pmm_agent_id=02ecd9e3-d7b8-4d94-9c75-060b8e6e3e84
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?service_id=/service_id/6984244c-0a18-4508-a219-3977e8fb01d0
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents?service_id=6984244c-0a18-4508-a219-3977e8fb01d0

### POST /v1/inventory/Agents/GetLogs - GET /v1/inventory/agents/{agent_id}/logs
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents/49bef198-299c-41b3-ba05-578defe63678/logs
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/agents/49bef198-299c-41b3-ba05-578defe63678/logs?limit=10

### POST /v1/inventory/Nodes/Get -> GET /v1/inventory/nodes/{node_id}
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/nodes/32c914d1-daf0-468a-aa9d-4ebb65ab2ee9

### POST /v1/inventory/Services/Get -> GET /v1/inventory/services/{service_id}
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/services/d4dfdccf-c07c-48a6-a101-b119b04d880f

### POST /v1/inventory/Services/List -> GET /v1/inventory/services
curl -s -X GET http://admin:admin@localhost:8080/v1/inventory/services

### POST /v1/inventory/Services/Change -> PUT /v1/inventory/services/{service_id} 
curl -s -X PUT -d '{"cluster": "test2","environment":"dev","replication_set":"main"}' http://admin:admin@localhost:8080/v1/inventory/services/d4dfdccf-c07c-48a6-a101-b119b04d880f
### add/update custom labels
curl -s -X PUT -d '{"custom_labels":{"values":{"env":"foo","bar":"123"}}}' http://admin:admin@localhost:8080/v1/inventory/services/d4dfdccf-c07c-48a6-a101-b119b04d880f
### remove a standard label and all custom labels
curl -s -X PUT -d '{"replication_set":"","custom_labels":{}}' http://admin:admin@localhost:8080/v1/inventory/services/d4dfdccf-c07c-48a6-a101-b119b04d880f

### POST /v1/inventory/Services/ListTypes -> POST /v1/inventory/services:getTypes
curl -s -X POST http://admin:admin@localhost:8080/v1/inventory/services:getTypes

### /v1/management/Service/Remove -> DELETE /v1/management/services/{service_id}
curl -s -X DELETE http://admin:admin@localhost:8080/v1/management/services/b7d3b87a-d366-4cb4-b101-03d68f73a7c0
### pmm-admin remove mongodb mongo-svc
### pmm-admin remove mongodb mongo-svc --service-id=/service_id/ed322782-e6fd-4ad9-8ee6-a7d47b62de41
### pmm-admin remove mongodb --service-id=/service_id/ed322782-e6fd-4ad9-8ee6-a7d47b62de41
