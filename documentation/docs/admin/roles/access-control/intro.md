# About label based access control (LBAC) in PMM

Access control in PMM allows you to manage access to data. By using access control you can restrict access to monitoring metrics and Query Analytics data. 

This is particularly important in environments where sensitive data is involved, and it helps ensure that only authorized users can access specific information, which is crucial for maintaining security and compliance.

## How LBAC works
PMM uses Prometheus label selectors to control access to metrics and Query Analytics data. 

Here's how LBAC works:
{.power-number}

1. Create roles with label selectors. For example `environment=prod` for a specific environment or `service_type=mysql` for specific databases.
2. Assign roles to users based on their responsibilities.
3. Users see only the metrics and (Query Analytics) QAN data that match their role's label selectors.

## Key benefits

- Granular permissions: Restrict access to specific services, environments, or regions.
- Enhanced security: Prevent unauthorized access to sensitive database metrics and query data.
- Compliance support: Meet regulatory requirements for data access control.
- Team-specific views: Allow teams to focus only on their relevant systems and queries.
- Simplified management: Manage access through roles instead of individual user permissions.

## Example scenarios

| User type | Possible role configuration | What they can see |
|-----------|---------------------------|------------------|
| DBA team lead | All services across environments | Complete monitoring data for all databases and queries |
| MySQL administrators | `service_type=mysql` | Only MySQL-related metrics and queries |
| Production support | `environment=production` | Only production environment metrics and queries |
| Regional team | `region=us-east` | Only metrics and queries from a specific region |

## Getting started with LBAC

To implement label-based access control in PMM:
{.power-number}

1. [Enable access control](enable_access_control.md) in your PMM settings
2. Learn about the [labels available for filtering](labels.md)
3. [Create access roles](create_roles.md) based on your organizational needs
4. Review common [use cases and examples](use_cases.md) for inspiration

!!! tip "Best practice"
    Start with broader access controls and refine them over time as you understand your organization's specific needs. Test LBAC behavior in both dashboards and QAN to ensure proper access control.

## Related topics

- [Manage PMM users](../../manage-users/index.md)
