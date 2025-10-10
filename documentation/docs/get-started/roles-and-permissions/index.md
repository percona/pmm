# Roles and permissions


Roles are the sets of permissions and configurations that determine which metrics a user can access.

Each PMM user is associated with a role that includes permissions. Permissions determine the privileges that a user has in PMM.

By creating roles, you can specify which data can be queried based on specific label criteria, for instance, allowing the QA team to view data related to test environments.

# About Access Control

!!! caution alert alert-warning "Caution"
    PMM Access Control is currently in [technical preview](../details/glossary.md#technical-preview) and is subject to change. We recommend that early adopters use this feature for testing purposes only.


Access control in PMM allows you to manage who has access to individual Prometheus (Victoria Metrics)  metrics based on **labels**. Thus, access management provides a standardized way of granting, changing, and revoking access to metrics based on the role assigned to the users.

The following topics are covered as part of access control:

- [Configure access control](configure_access_roles.md)
- [Labels for access control](lbac.md)
- [Create access roles](access_roles.md)
- [Use case](use_case.md)