---
title: Create user accounts
slug: create-user-accounts
categorySlug: pmm-server-maintenance
parentDocSlug: pmm-server-user-accounts
order: 1
---

### Create multiple user accounts

PMM Server will start up with a single user account, the administrator account.
This account should not be for general use, because it allows complete control of
the server and access to all the data, including sensitive.

Similar to the management of the [admin account](ref:change-admin-password), it is
possible to manage other user accounts via the API. Waiting for readiness and
using `netrc` files with `curl` are relevant for API usage in general and are
thus relevant here.

### Create a single user account

When creating user accounts it is necessary to have admin privileges for Grafana,
we will assume the use of the default administrator accounts in the examples that
follow.

The following example will create a Grafana admin user that can make certain
administrative changes to the system, but does not have full administrative access.

```shell
$ cat <<EOF >/tmp/data.json
{
    "name": "Grafana Admin",
    "login": "grafana-admin",
    "password": "ChangeME123!"
}
EOF

$ curl --silent -X POST --netrc \
    --header "Content-Type: application/json" \
    --data @/tmp/data.json  https://127.0.0.1/graph/api/admin/users
{"id":2,"message":"User created"}
```

This can be acheived with Ansible with a task such as:
```yaml
- name: Create a new user
  uri:
    url: https://127.0.0.1/graph/api/admin/users
    method: POST
    user: admin
    password: admin
    force_basic_auth: true
    validate_certs: true
    body_format: json
    body:
      name: 'Grafana Admin'
      login: 'grafana-admin'
      password: 'ChangeME123!'
    status_code:
    - 201
    - 412
  no_log: true
```
The accepted status codes here will allow the task to pass when the user already exists (412) as well as when the account is created (201).

### Create API tokens

Instead of login credentials, you can create API tokens, which are associated with an organisation and can be used to create dashboards, or other components.

Here is an example that creates an admin API token for the `grafana-admin` user that
was just created:
```sh
$ curl --silent -X POST --netrc \
    --header "Content-Type: application/json" \
    --data '{"name": "grafana-admin-token", "role": "Admin"}'  https://127.0.0.1/graph/api/auth/keys
{"id":3,"name":"grafana-admin-token","key":"eyJrIjoiQjYxa05xY3doU1dDczdudnppdnJVeUdjS3k0Y05vMW0iLCJuIjoiZ3JhZmFuYS1hZG1pbi10b2tlbiIsImlkIjoxfQ=="}
```

This can be achieved in Ansible with a task such as:
```yaml
- name: Create a new API token
  uri:
    url: https://127.0.0.1/graph/api/auth/keys
    method: POST
    user: admin
    password: admin
    force_basic_auth: true
    validate_certs: true
    body_format: json
    body:
      name: 'grafana-admin-token'
      role: 'Admin'
  no_log: true
```

### Combining this all together to create multiple accounts

For simplicity, we will only be using Ansible for this example.
```yaml
- name: Create a new user
  uri:
    url: https://127.0.0.1/graph/api/admin/users
    method: POST
    user: admin
    password: admin
    force_basic_auth: true
    validate_certs: true
    body_format: json
    body:
      name: '{{ item.user }}'
      login: '{{ item.login }}'
      password: '{{ item.password }}'
    status_code:
    - 200
    - 201
    - 412
  no_log: true
  loop:
  - user: User 1
    login: user1
    password: 'ChangeME123!'
  - user: User 2
    login: user2
    password: 'ChangeME123!'

- name: Create a new API token
  uri:
    url: https://127.0.0.1/graph/api/auth/keys
    method: POST
    user: admin
    password: admin
    force_basic_auth: true
    validate_certs: true
    body_format: json
    body:
      name: '{{ item.login }}-{{ item.role }}'
      role: '{{ item.role }}'
  no_log: true
  loop:
  - login: user1
    role: Viewer
  - login: user2
    role: Viewer
```
