# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

### Advisor insights, schedules and custom checks

Advisors now have four pages:

- **Insights** keeps the findings of every run for 30 days by default, and you can filter them and email them to any address.
- **Catalog** lists the built-in checks and the checks that you write. You can clone a built-in check, and test a check on a service before you save it.
- **Run history** shows what each run covered and which schedule started it.
- **Scheduler** runs Advisor checks on schedules instead of fixed intervals. For each schedule, you choose the days and time in UTC, the checks and services, and who gets the email report.

Links to the previous Advisor pages open the new pages.

Only Admins can create, edit, test and delete checks, because a check's queries run with the privileges of the PMM monitoring user. Editors can run checks, turn them on or off, and manage schedules.

To get started, see [Advisor details](../advisors/advisors.md), [Schedule Advisor checks](../advisors/schedule-advisor-checks.md) and [Develop Advisor checks](../advisors/develop-advisor-checks.md).

#### What changes when you upgrade

The upgrade changes the following:

- **Run intervals are removed.** The **Rare**, **Standard** and **Frequent** run intervals, the interval of each check, and the runs at PMM Server startup and when you turn on Advisors are gone. Instead, the **All checks daily** schedule runs every check on every service at 06:00 UTC. If you changed the intervals, recreate them as schedules.
- **Checks that you turned off are enabled again.** To turn them off, go to **Advisors > Catalog** and use the **Status** switch. For details, see [Enable/Disable](../advisors/advisors.md#enabledisable).
- **The Advisor dashboard panels are removed.** The **Advisor Insights** panel on the **Environments Overview** dashboard and the **Failed advisors** panel on the **Home** dashboard are gone. Review Advisor results on **Advisors > Insights** instead.
- **Some Advisor API endpoints and fields are removed.** To list checks, use `GET /v1/advisors/checks` instead of `GET /v1/advisors`. To read results, use `GET /v1/advisors/insights` instead of `GET /v1/advisors/failedServices` and `GET /v1/advisors/checks/failed`. The `advisor_run_intervals` setting and the `interval` field of checks are removed. To control when checks run, use the `/v1/advisors/schedules` endpoints.

## 🔒 Security updates

## 📈 Improvements

- [PMM-15239](https://perconadev.atlassian.net/browse/PMM-15239): The **Alert Status** page now groups alert severities into four bands: **Critical**, **Error**, **Warning** and **Info**. Each band has its own icon and label, so you can tell severities apart without relying on color. The **Severity** filter and sort work by band, and the alert details show the original level, for example **Reported as Emergency**. Alert labels stay the same. For how levels map to bands, see [Alert rules](../alert/alert_rules.md).
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).
- [PMM-14109](https://perconadev.atlassian.net/browse/PMM-14109): Advisors now keep a history of insights and runs, run checks on schedules instead of fixed intervals, and support custom checks. The upgrade removes the run intervals and some Advisor API endpoints, and enables every check again. For details, see [Advisor insights, schedules and custom checks](#advisor-insights-schedules-and-custom-checks).

## ✅ Fixed issues

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
