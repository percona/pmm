# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15182](https://perconadev.atlassian.net/browse/PMM-15182): PMM Server no longer ships vmalert, an unused leftover from the old Integrated Alerting. Alerting is unaffected and stays in PMM Alerting. If you added your own rule files under `/srv/prometheus/rules` or through `rule_files` in `prometheus.base.yml`, they are no longer evaluated: recreate them as [Grafana alert or recording rules](../alert/alert_rules.md).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
