# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15683](https://perconadev.atlassian.net/browse/PMM-15683): **MongoDB > Operations > Settings** now shows the range each number field accepts, for example **A whole number from 1 to 8**, and refuses a value outside it before you save. When PMM Extensions refuses a save or reset, the page lists one readable line per field, named by its label, instead of raw JSON. **Period** no longer offers `seconds`, which the scan schedule never accepted. For details, see [Operations for MongoDB](../use/mongodb-operations.md#change-the-scan-schedule).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
