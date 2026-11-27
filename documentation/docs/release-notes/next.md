# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15881](https://perconadev.atlassian.net/browse/PMM-15881): PMM Server now includes the official open source release of VictoriaMetrics instead of a PMM-specific build, so PMM can ship VictoriaMetrics fixes without waiting for that build. **Breaking change:** the open source release doesn't include downsampling and doesn't read Prometheus data files. PMM ignores `VM_downsampling_period` and `VM_prometheusDataPath` and logs a warning at startup. If you use downsampling, new metrics stay at full resolution and need more disk space, and data that is already downsampled stays downsampled. To limit disk use or to import Prometheus data files, see [Downsampling and Prometheus data files](../reference/third-party/victoria.md#downsampling-and-prometheus-data-files).

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
