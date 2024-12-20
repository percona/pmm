---
title: Change the administrator's password
slug: change-admin-password
categorySlug: pmm-server-maintenance
parentDocSlug: pmm-server-user-accounts
order: 0
---

## Changing the admin password

Your new PMM Server will start up with a default set of credentials for the administrator account. When you first login via the UI, you will be prompted to change this to a new password in order to secure the account.

When automating the deployment and/or management of your PMM Server, it is preferable to use the available APIs instead of relying upon human interaction.

The [authentication](authentication) overview explains the different ways that you can programmatically access the API, using either [basic](authentication#basic-http-authentication), or [token-based](authentication#bearer-authentication) authentication. **Note:** for basic authentication, you can use the same approach for standard credentials if you haven't yet created a token.

**Note** Examples using cURL will be shown with the `--netrc` argument, which allows credentials to be hidden from view in the processlist and shell history. Should you wish to use credentials in the command instead then substitute `--netrc` for `--basic --user '<user>:<password>'`

Here is an example `.netrc`:
```
machine 127.0.0.1
login admin
password admin
```

### Check that the server is ready for you

When the server starts up, there are a number of tasks that execute before the server and API are reliably available to the user. You can check whether the server is ready using the dedicated endpoint, [`/v1/readyz`](https://percona-pmm.readme.io/reference/readiness):

```shell
$ curl --silent https://127.0.0.1/v1/readyz
{}
```

If the server is not yet ready then you will see a response such as:
```json
{
  "error": "PMM Server is not ready yet.",
  "code": 13,
  "message": "PMM Server is not ready yet."
}
```

You can check this using Ansible with a task such as:
```yaml
- name: Wait for PMM Server to be ready
  uri:
    url: https://127.0.0.1/v1/readyz
    method: GET
    validate_certs: true
    headers:
      Content-Type: application/json
    status_code:
      - 200
      - -1
  register: pmm_api_status
  until: (pmm_api_status.status is defined) and (pmm_api_status.status == 200)
  retries: 10
  delay: 10
```

### Changing the password following initial installation

**Caution:** Once you have changed the password, you need to use the new password from then on. Should you be doing this for an account that is used elsewhere then you may find that the account gets blocked due to too many failed login attempts. You should disconnect any clients using the same account before proceeding to avoid such issues.

The [payload](https://grafana.com/docs/grafana/latest/http_api/user/#change-password) for changing a user's password is:
```json
{
  "oldPassword": "xxx",
  "newPassword": "yyy",
  "confirmNew": "yyy"
}
```

Here is an example that changes the password from `admin` to `notAdminAnymore`:
```shell
$ cat <<EOF > /tmp/data.json
{
  "oldPassword": "admin",
  "newPassword": "notAdminAnymore",
  "confirmNew": "notAdminAnymore"
}
EOF

$ curl --silent -X PUT --netrc \
    --header "Content-Type: application/json" \
    --data @/tmp/data.json https://127.0.0.1/graph/api/user/password
{"message":"User password changed"}
```

This can be achieved in Ansible with a task such as:
```yaml
- name: Change the admin password
  uri:
    url: https://127.0.0.1/graph/api/user/password
    method: PUT
    user: admin
    password: admin
    force_basic_auth: true
    validate_certs: true
    body_format: json
    body:
      oldPassword: 'admin'
      newPassword: 'notAdminAnymore'
      confirmNew: 'notAdminAnymore'
  no_log: true
```
