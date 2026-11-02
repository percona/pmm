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

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated [Nomad](../reference/nomad.md) to 2.1.1, which fixes [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the `go-jose/v4` finding for [CVE-2026-116587](https://nvd.nist.gov/vuln/detail/CVE-2026-116587). Security scanners still report CVE-2026-116587 for Nomad's `go-jose/v3`, which has no fixed release.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
