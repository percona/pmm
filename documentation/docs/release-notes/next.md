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

## ✅ Fixed issues

## 🔶 Known issues

### Older annotations can be missing on dashboards that cover many nodes and services ([PMM-15663](https://perconadev.atlassian.net/browse/PMM-15663))

When a dashboard's selection covers 500 or more nodes and services, PMM reads only the newest 100,000 annotations in the time range. This happens, for example, when you select **All** on a large fleet. If the time range holds more annotations than that, the older ones aren't shown. This includes annotations set globally.

Dashboards show up to 100 annotations, so this only matters when few of the newest 100,000 belong to the dashboard's nodes and services. To see the older annotations, shorten the time range, or select fewer than 500 nodes and services. For more about annotations, see [Annotation in dashboards](../use/dashboards-panels/annotate/annotate.md).

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
