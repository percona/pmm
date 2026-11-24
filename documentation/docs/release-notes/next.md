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

- [PMM-15003](https://perconadev.atlassian.net/browse/PMM-15003): The **Inventory > Add service** pages now run in native PMM UI, like [**Inventory**](3.11.0.md#inventory-page-now-in-native-pmm-ui) since PMM 3.11.0, so adding a service looks and works like the rest of PMM. Your existing links to these pages keep working.

## ✅ Fixed issues

- [PMM-15003](https://perconadev.atlassian.net/browse/PMM-15003): Fixed two issues when you add a service on **Inventory > Add service**. If you chose HTTPS for an external service, PMM still collected its metrics over HTTP. If you added a MongoDB service with TLS, PMM ignored the **TLS certificate password** that you entered. Both services now connect the way you set them up.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
