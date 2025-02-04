# Service accounts authentication

!!! caution alert alert-warning "Deprecation notice"
    Starting with version 3, PMM no longer uses API keys as the primary method for controlling access to the PMM Server components and resources. Instead, PMM is now leveraging Grafana service accounts, which have limited scopes and offer enhanced security compared to API keys.

### Automatic migration of API keys

When you install PMM v3.x, any existing API keys will be seamlessly converted to service accounts with corresponding service tokens. For more information about the migration, see [Migrate PMM 2 to PMM 3](../pmm-upgrade/migrating_from_pmm_2.md).

Service accounts in PMM provide a secure and efficient way to manage access to the PMM Server and its resources. They serve as a replacement for the basic authentication and API keys used in previous versions of PMM (v.2 and earlier).

With service accounts, you can:

- control access to PMM Server components and resources.
- define granular permissions for various actions.
- create and manage multiple access tokens for a single service account.

Creating multiple tokens for the same service account is beneficial in the following scenarios:

- when multiple applications require the same permissions but need to be audited or managed separately. By assigning each application its own token, you can track and control their actions individually.
- when a token becomes compromised and needs to be replaced. Instead of revoking the entire service account, you can rotate or replace the affected token without disrupting other applications using the same service account.
- when you want to implement token lifecycle management. You can set expiration dates for individual tokens, ensuring that they are regularly rotated and reducing the risk of unauthorized access.

## Service Account name management

To prevent node registration failures, PMM automatically manages service account names that exceed 200 characters using a `{prefix}_{hash}` pattern. For example, a very long service account name will be automatically shortened while maintaining uniqueness:

- **original**: `very_long_mysql_database_server_in_production_environment_with_specific_location_details...`
- **shortened**: `very_long_mysql_database_server_in_prod_4a7b3f9d`

## Generate a service account and token

PMM uses Grafana service account tokens for authentication. These tokens are randomly generated strings that serve as alternatives to API keys or basic authentication passwords.

Here's how to generate a service account token:

1. Log into PMM.
2. From the side menu, click **Administration > Users and access**.
3. Click on the **Service accounts** card.
4. Click **Add service account**. Specify a unique name for your service account, select a role from the drop-down menu, and click **Create** to display your newly created service account.
5. Click **Add service account token**.
6. In the pop-up dialog, provide a name for the new service token, or leave the field empty to generate an automatic name.
7. Optionally, set an expiration date for the service account token. PMM cannot automatically rotate expired tokens, which means and you will need to manually [update the PMM-agent configuration file](../use/commands/pmm-agent.md) with a new service account token. Permanent tokens, on the other hand, remain valid indefinitely unless specifically revoked.
8. Click **Generate Token**. A pop-up window will display the new token, which usually has a *glsa_* prefix.
9. Copy your service token to the clipboard and store it securely.
Now you can use your new service token for authentication in PMM API calls or in your [pmm-agent configuration](../use/commands/pmm-agent.md).

## Authenticate

You can authenticate your request using the HTTPS header.

!!! caution alert alert-warning "Important"
    Use the `-k` or `--insecure` parameter to force cURL to ignore invalid and self-signed SSL certificate errors. The option will skip the SSL verification process, and you can bypass any SSL errors while still having SSL-encrypted communication. However, using the `--insecure`  parameter is not recommended. Although the data transfer is encrypted, it is not entirely secure. For enhanced security of your PMM installation, you need valid SSL certificates. For information on validating SSL certificates, refer to: [SSL certificates](../how-to/secure.md).

```sh
curl -H "Authorization: Bearer <service_token>" https://127.0.0.1/v1/version
```

## Use a service token in basic authentication

You can include the service token as a query parameter in a REST API call using the following format. Replace YOUR_SERVICE_TOKEN with the actual service token you obtained in step 9.


**Example**
```sh
curl -X GET https://service_token:SERVICE_TOKEN@localhost/v1/version
```

## Use a service token in Bearer authentication (HTTP header)
You can also include the service token in the header of an HTTP request for authentication. To do this, replace `SERVICE_TOKEN` with the actual service token you obtained in step 9.

**Example**
```shell
curl -X GET -H 'Authorization: Bearer SERVICE_TOKEN' \
  -H 'Content-Type: application/json' https://127.0.0.1/v1/version
```