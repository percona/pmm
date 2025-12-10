---
title: Authentication
slug: authentication
parentSlug: welcome
order: 1
---

## Authentication

PMM Server's user authentication is built on top of and is compatible with [Grafana authentication](https://grafana.com/docs/grafana/latest/auth/grafana/).

Starting with PMM v3, authentication uses **Grafana service accounts** with limited scopes and enhanced security. Service accounts provide a secure and efficient way to manage access to PMM Server components and resources.

## Service accounts (recommended)

Service accounts are the primary and recommended authentication method for PMM v3.x. They replace the basic authentication and API keys used in PMM 2.

### Benefits of service accounts

With service accounts, you can:

- Control access to PMM Server components and resources
- Define granular permissions for various actions
- Create and manage multiple access tokens for a single service account
- Implement token lifecycle management with expiration dates

Creating multiple tokens for the same service account is beneficial when:

- Multiple applications require the same permissions but need to be audited or managed separately. By assigning each application its own token, you can track and control their actions individually.
- A token becomes compromised and needs to be replaced. Instead of revoking the entire service account, you can rotate or replace the affected token without disrupting other applications using the same service account.
- You want to implement token lifecycle management. You can set expiration dates for individual tokens, ensuring they are regularly rotated and reducing the risk of unauthorized access.

### Service account name management

To prevent node registration failures, PMM automatically manages service account names that exceed 200 characters using a `{prefix}_{hash}` pattern. For example:

- **Original**: `very_long_mysql_database_server_in_production_environment_with_specific_location_details...`
- **Shortened**: `very_long_mysql_database_server_in_prod_4a7b3f9d`

### Generate a service account and token

PMM uses Grafana service account tokens for authentication. These tokens are randomly generated strings that serve as secure alternatives to basic authentication passwords.

To generate a service account token:

1. Log into PMM.
2. From the side menu, click **Administration > Users and access**.
3. Click on the **Service accounts** card.
4. Click **Add service account**. Specify a unique name for your service account, select a role from the drop-down menu, and click **Create** to display your newly created service account.
5. Click **Add service account token**.
6. In the pop-up dialog, provide a name for the new service token, or leave the field empty to generate an automatic name.
7. Optionally, set an expiration date for the service account token. PMM cannot automatically rotate expired tokens, which means you will need to manually [update the PMM-agent configuration file](../use/commands/pmm-agent.md) with a new service account token. Permanent tokens remain valid indefinitely unless specifically revoked.
8. Click **Generate Token**. A pop-up window will display the new token, which usually has a `glsa_` prefix.
9. Copy your service token to the clipboard and store it securely.

Now you can use your new service token for authentication in PMM API calls or in your [pmm-agent configuration](../use/commands/pmm-agent.md).

## Authenticating with service tokens

!!! caution alert alert-warning "Important"
    Use the `-k` or `--insecure` parameter to force cURL to ignore invalid and self-signed SSL certificate errors. The option will skip the SSL verification process, and you can bypass any SSL errors while still having SSL-encrypted communication. However, using the `--insecure` parameter is not recommended. Although the data transfer is encrypted, it is not entirely secure. 
    
    For enhanced security of your PMM installation, you need valid SSL certificates. For information on validating SSL certificates, see [SSL certificates](../admin/security/ssl_encryption.md).

### Bearer authentication with service tokens

Include the service token in the Authorization header of an HTTP request:

```shell
curl -H "Authorization: Bearer glsa_Fp0ggev31R58ueNJbJgYw7fIGfO3yKWH_746383ab" \
  -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```

### Basic authentication with service tokens

You can also use the service token in basic authentication. Use `service_token` as the username and the service token as the password:

```shell
curl -X GET https://service_token:glsa_Fp0ggev31R58ueNJbJgYw7fIGfO3yKWH_746383ab@127.0.0.1/v1/version
```

## Protecting credentials

Credentials should not be exposed in shell history or process lists. Here are recommended practices:

### Disable history

**Bash:**
```shell
set +o history
```

**Zsh:**
```zsh
SAVEHIST=0
```

### Using --netrc with curl

You can store credentials in a `~/.netrc` file and reference it with the `--netrc` option. This keeps credentials out of shell history and visible commands.

Example `~/.netrc`:
```
machine 127.0.0.1
login service_token
password glsa_Fp0ggev31R58ueNJbJgYw7fIGfO3yKWH_746383ab
```

Use it with curl:
```shell
curl --netrc -X GET -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```

To use a different file:
```shell
curl --netrc --netrc-file ~/.netrc-pmm -X GET -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```

## Legacy authentication methods

!!! caution alert alert-warning "Deprecation notice"
    The following authentication methods are deprecated in PMM v3.x. They continue to work for backward compatibility but should not be used for new implementations. Use service accounts instead.

### Basic HTTP authentication (Deprecated)

Basic authentication is a simple way to authenticate a user. An API request must contain an Authorization header with Base64-encoded credentials:

```shell
curl -X GET -u admin:admin \
  -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```

Where the credentials are formatted as `username:password` and Base64-encoded:

```shell
echo -n admin:admin | base64
```

### API keys (Deprecated)

!!! info "Migration in PMM v3"
    When you upgrade to PMM v3.x, any existing API keys are seamlessly converted to service accounts with corresponding service tokens. For more information, see [Migrate PMM 2 to PMM 3](../pmm-upgrade/migrating_from_pmm_2.md).

API keys are no longer the primary authentication method and have been replaced by service accounts. If you have existing API keys from PMM v2.x, they will be automatically migrated to service tokens.

Legacy API key example (not recommended):
```shell
curl -X GET -H 'Authorization: Bearer eyJrIjoiUXRkeDNMS1g1bFVyY0tUj1o0SmhBc3g4QUdTRVAwekoiLCJuIjoicG1tLXRlc3QiLCJpZCI6MX0=' \
  -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```

You can use API keys in basic authentication as well (not recommended):
```shell
curl -X GET -u api_key:eyJrIjoiUXRkeDNMS1g1bFVyY0tUj1o0SmhBc3g4QUdTRVAwekoiLCJuIjoicG1tLXRlc3QiLCJpZCI6MX0= \
  -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```