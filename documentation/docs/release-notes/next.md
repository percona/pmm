# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source database monitoring, management, and observability solution for MySQL, PostgreSQL, MongoDB, Valkey and Redis. PMM empowers you to:

- monitor the health and performance of your database systems
- identify patterns and trends in database behavior
- diagnose and resolve issues faster with actionable insights
- manage databases across on-premises, cloud, and hybrid environments

## 📋 Release summary

## ✨ Release highlights

### Smarter Advisors that run on schedule, email results and suggest fixes

You no longer need to open PMM to find out what Advisors found. [Schedule checks](../advisors/schedule-advisor-checks.md) to run when it suits your team, on the services that matter to you, and PMM can [email you a report](../advisors/advisors.md#get-advisor-results-by-email) after each run.

For anything the built-in checks don't cover, [write your own checks](../advisors/develop-advisor-checks.md) and test them on a service before you save them.


When you need to follow up on an issue, the new **Insights** page shows which services it affects, when it was found and, for most built-in checks, how to fix it:


![Details of a failed Advisor insight, with the affected service, check time, Read more link and outcome](../images/advisor-insights.png)

#### What changes when you upgrade

Upgrading to PMM 3.11.0 changes your existing Advisor settings, dashboards and API calls:

- Schedules replace run intervals. By default, the **All checks daily** schedule runs every check at 06:00 UTC. If you used custom intervals, [recreate them as schedules](../advisors/schedule-advisor-checks.md#create-a-schedule).
- All checks run again, including the ones that you turned off. [Turn them off again](../advisors/advisors.md#turn-off-checks-that-dont-apply) in **Advisors > Catalog**.
- **Advisors > Insights** replaces the Advisor panels on the **Home** and **Environments Overview** dashboards. Review findings there instead.
- New endpoints replace some Advisor API endpoints. If your scripts call the Advisor API, switch to these endpoints:
    - `GET /v1/advisors/checks` instead of `GET /v1/advisors`
    - `GET /v1/advisors/insights` instead of `GET /v1/advisors/failedServices` and `GET /v1/advisors/checks/failed`
    - The `/v1/advisors/schedules` endpoints instead of the `advisor_run_intervals` setting and the `interval` field of checks

## 🔒 Security updates

## 📈 Improvements

- [PMM-15239](https://perconadev.atlassian.net/browse/PMM-15239): The **Alert Status** page now groups alert severities into four bands: **Critical**, **Error**, **Warning** and **Info**. Each band has its own icon and label, so you can tell severities apart without relying on color. The **Severity** filter and sort work by band, and the alert details show the original level, for example **Reported as Emergency**. Alert labels stay the same. For how levels map to bands, see [Alert rules](../alert/alert_rules.md).
- [PMM-15805](https://perconadev.atlassian.net/browse/PMM-15805): The **Services** tab on **MongoDB > Operations > Fleet** now shows why a service is failing. The **Collected** column shows the reason, for example **Database unreachable**, and the service details show the full error, what to do about it, and a link to the scan that failed. A **Down** service whose `mongod` or `mongos` wasn't running at its last scan shows **not running**. The **Clusters** tab now warns when scan results can't be read. For details, see [Operations for MongoDB](../use/mongodb-operations.md#scans).
- [PMM-14109](https://perconadev.atlassian.net/browse/PMM-14109): Advisors now email a report after each run, keep a history of insights and runs, [run checks on schedules](#smarter-advisors-that-run-on-schedule-email-results-and-suggest-fixes) instead of fixed intervals, and support custom checks. The upgrade removes the run intervals and some Advisor API endpoints, and enables every check again.

## ✅ Fixed issues

- [PMM-15654](https://perconadev.atlassian.net/browse/PMM-15654): Fixed the [MongoDB Sharded Cluster Summary](../reference/dashboards/dashboard-mongodb-cluster-summary.md) dashboard hiding clusters added without `--environment` when **Environment** is set to **All** and another MongoDB service has an environment set. These clusters now appear under **All**.

- [PMM-7](https://perconadev.atlassian.net/browse/PMM-7): Updated the Grafana ClickHouse data source plugin to v4.23.4, built with Go 1.27.3. This fixes the high-severity Go vulnerabilities that image scanners reported in v4.21.3, including [CVE-2026-78669](https://nvd.nist.gov/vuln/detail/CVE-2026-78669) and the standard library findings listed in the [PMM 3.10.0 release notes](3.10.0.md#go-standard-library-vulnerabilities-in-the-clickhouse-data-source).

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
