
```bash
// POST /actions/v1/action:start
	body = {
		MongoDBExplain: {
			action_name: MongoDBExplain
		}
	}
```

| Current                        | Migrate to                            | Comments                     |
| ------------------------------ | ------------------------------------- | ---------------------------- |
| POST /v1/updates/Check         | GET /v1/updates                       |                              |
| POST /v1/updates/Start         | POST /v1/updates:start                |                              |
| POST /v1/updates/Status        | GET /v1/updates/status?log_offset=200 | "auth_token" - pass via headers |
| POST /v1/management/HAProxy/Add | POST /management/v1/services/HAProxy  |                              |
| POST /v1/management/Service/Remove | DELETE /management/v1/services/{id} | {service_id} or {service_name} |


// POST /v1/management/Service/Remove => DELETE /management/v1/services/{id}
