# Standard role permissions

Roles are the sets of permissions and configurations that determine which metrics a user can access in Percona Monitoring and Management (PMM).

Each PMM user is associated with a role that includes permissions. Permissions determine the privileges that a user has in PMM.

PMM provides two methods of access control: standard roles (Viewer, Editor, Admin) that determine feature-level permissions, and label-based access control that allows administrators to create custom roles to specify which data can be queried based on specific label criteria, for instance, allowing the QA team to view data related only to test environments.

For more granular data access control, see [Labels for access control](../roles/access-control/intro.md) which allows you to restrict which metrics users can query based on labels.

## Role types in PMM

PMM inherits its basic role structure from [Grafana](https://grafana.com/docs/grafana/latest/administration/roles-and-permissions/) but with customizations specific to database monitoring. PMM has three main role types:

- **Admin**: Has access to all resources and features within a PMM instance. This role can manage all aspects of PMM including users, teams, data sources, dashboards, and server settings.
- **Editor**: Can view and edit dashboards, create custom visualizations, work with alerts, and manage specific configurations. Editors cannot modify server-wide settings or manage users.
- **Viewer**: Has read-only access to monitoring data and dashboards. Viewers can query data but cannot make changes to configurations.

## Default role assignment

When a user signs in to PMM for the first time and has no role assigned, they are automatically assigned the default role. Administrators can configure which role is used as the default through the access control settings.

## Dashboard permissions

Dashboard creators in PMM automatically get Admin permissions for the dashboards they create. Folder permissions cascade to all dashboards within that folder.

## Permission matrix

=== "Dashboard & Monitoring"
    | Permission | Viewer | Editor | Admin |
    |------------|--------|--------|-------|
    | View dashboards | ✓ | ✓ | ✓ |
    | Add, edit, delete dashboards | ✗ | ✓ | ✓ |
    | Add, edit, delete folders | ✗ | ✓ | ✓ |
    | View playlists | ✓ | ✓ | ✓ |
    | Add, edit, delete playlists | ✗ | ✓ | ✓ |
    | Access Explore | ✗ | ✓ | ✓ |
    | Query data sources | ✓ | ✓ | ✓ |
    | View Query Analytics | ✓ | ✓ | ✓ |
    | View Insights | ✓ | ✓ | ✓ |

=== "Alerting & Advisors"
    | Permission | Viewer | Editor | Admin |
    |------------|--------|--------|-------|
    | View Alert Rules | ✓ | ✓ | ✓ |
    | Add, edit, delete Alert Rules | ✗ | ✓ | ✓ |
    | View Fired Alerts | ✓ | ✓ | ✓ |
    | Silence Alerts | ✗ | ✓ | ✓ |
    | View Alert Templates | ✗ | ✓ | ✓ |
    | Create Alerts from Templates | ✗ | ✓ | ✓ |
    | Add, edit, delete Alert Templates | ✗ | ✓ | ✓ |
    | View Advisors | ✗ | ✓ | ✓ |
    | Run, disable, edit Advisors | ✗ | ✗ | ✓ |
    | Run Advisor Checks | ✗ | ✗ | ✓ |

=== "Configuration & Management"
    | Permission | Viewer | Editor | Admin |
    |------------|--------|--------|-------|
    | View Inventory | ✓ | ✓ | ✓ |
    | Add, edit, delete Services | ✗ | ✗ | ✓ |
    | View Actions | ✓ | ✓ | ✓ |
    | Execute Actions | ✓ | ✓ | ✓ |
    | View Server Settings | ✓ | ✓ | ✓ |
    | Modify Server Settings | ✗ | ✗ | ✓ |
    | Add, edit, delete Users | ✗ | ✗ | ✓ |
    | Add, edit, delete Teams | ✗ | ✗ | ✓ |
    | View Backups | ✗ | ✗ | ✓ |
    | Manage Backups | ✗ | ✗ | ✓ |
    | View Update Status | ✓ | ✓ | ✓ |
    | Start Updates | ✗ | ✗ | ✓ |

=== "Data sources"
    | Permission | Viewer | Editor | Admin |
    |------------|--------|--------|-------|
    | View Data Sources | ✓ | ✓ | ✓ |
    | Add, edit, delete Data Sources | ✗ | ✗ | ✓ |
    | Configure Data Source Access | ✗ | ✗ | ✓ |

=== "API access"
    | API Path | Minimum role required | Purpose |
    |----------|----------------------|---------|
    | `/v1/alerting` | Viewer | Access alert information |
    | `/v1/advisors` | Editor | Access advisor functionality |
    | `/v1/advisors/checks` | Admin | Run advisor checks |
    | `/v1/actions/` | Viewer | View and execute actions |
    | `/v1/backups` | Admin | Manage backups |
    | `/v1/inventory/` | Admin | Manage inventory items |
    | `/v1/inventory/services:getTypes` | Viewer | View service types |
    | `/v1/management/` | Admin | Server management functions |
    | `/v1/management/Jobs` | Viewer | View management jobs |
    | `/v1/server/updates` | Viewer | Check for updates |
    | `/v1/server/updates:start` | Admin | Start update process |
    | `/v1/server/settings/readonly` | Viewer | View read-only settings |
    | `/v1/server/settings` | Admin | Configure server settings |
    | `/v1/server/settings/readonly` | Viewer | View read-only settings |
    | `/v1/platform:` | Admin | Platform management |
    | `/v1/platform/` | Viewer | Platform information |
    | `/v1/qan` | Viewer | Query Analytics |
    | `/v1/users` | Viewer | View user information |

