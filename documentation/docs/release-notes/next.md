# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

## ✅ Fixed issues

- [PMM-15534](https://perconadev.atlassian.net/browse/PMM-15534): Fixed an issue where access roles didn't apply to requests made with a service account token. With access control enabled, a token now gets only the metrics and Query Analytics data that the roles of its service account allow. A service account with no role gets the default role when one of its tokens first makes a request that access roles filter, such as a metrics or Query Analytics query. If you restricted the default role, check the service account tokens that you created, including API keys converted from PMM 2, before you upgrade, and assign their service accounts the roles they need. PMM Client on registered nodes keeps working as before, whatever the default role. For details, see [Assign access roles to service accounts](../admin/roles/access-control/assign_roles.md#assign-access-roles-to-service-accounts).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
