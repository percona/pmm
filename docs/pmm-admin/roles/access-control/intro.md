# About access control in PMM

!!! caution alert alert-warning "Caution"
    PMM Access Control is currently in [technical preview](../../../reference/glossary.md#technical-preview) and is subject to change. We recommend that early adopters use this feature for testing purposes only.

Access control in PMM allows you to manage who has access to individual Prometheus (Victoria Metrics)  metrics based on **labels**. Thus, access management provides a standardized way of granting, changing, and revoking access to metrics based on the role assigned to the users.

The following topics are covered as part of access control:

- [Configure access control](config_access_cntrl.md)
- [Labels for access control](labels.md)
- [Create access roles](create_roles.md)
- [Use case](usecase.md)