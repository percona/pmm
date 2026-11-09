---
title: Overview
slug: pmm-advisors
category:
  uri: advisor-api
position: 0
---


This section describes APIs to deal with Percona [Advisors](https://docs.percona.com/percona-monitoring-and-management/3/advisors/advisors.html), and [Advisors checks](https://docs.percona.com/percona-monitoring-and-management/3/advisors/advisor-details.html).

- The [List of Problems Detected by Advisors](getfailedchecks) API endpoint offers detailed insights into potential infrastructure issues identified by Advisors.
- [List Percona Advisors](ref:listadvisors) lists all Advisors available for your PMM instance.
- [List Advisor Checks](ref:listadvisorchecks) lists all Advisor Checks available in your PMM.
- [Changing Advisor Checks](ref:changeadvisorchecks) will help you automate Advisor Checks, i.e.:
  - Enable/Disable Advisor Checks
  - Change Advisor Check execution interval
