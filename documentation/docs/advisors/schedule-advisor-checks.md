# Schedule Advisor checks

Advisor schedules decide when PMM runs Advisor checks automatically, which checks and services each run covers, and who gets the email report. You manage them on the **Advisors > Scheduler** page, which lists every schedule with its next and last run.

PMM comes with one schedule, **All checks daily**, which runs every check on every service each day at 06:00 UTC. For details, see [Default schedule](#default-schedule).

## Prerequisites

Before you start, make sure that:

- The **Advisors** option is enabled under **Configuration > Settings > Advisors**.
- You have the **Editor** or **Admin** role. For details, see [Permissions](#permissions).
- To get email reports, PMM can send email and **Advisor notifications** are turned on. For details, see [Email notifications](advisors.md#email-notifications).

## Create a schedule

To create a schedule:
{.power-number}

1. Go to **Advisors > Scheduler** and click **New schedule**.
2. In **Name**, enter a name that no other schedule uses, up to 100 characters. The name appears in **Run history** and in the subject of the email report.
3. Under **When**, select the **Days** and enter the **Time (UTC)** as `HH:MM`. To select several days at once, click **every day**, **weekdays** or **weekends** next to **Set to**. The preview shows the next runs in UTC and in your time zone.
4. Under **Checks**, select **All checks** or **Selected checks**. For **Selected checks**, pick the checks in the list. To narrow the list, use **Search checks** and the **Category** filter.
5. Under **Services**, select **All services** or **Selected services**. For **Selected services**, pick the services in the list. To narrow the list, use the search field and the **Environment** filter.
6. Under **Email report**, choose who gets the report and which severities it covers. For details, see [Email reports](#email-reports).
7. Click **Create schedule**. A message confirms that PMM created the schedule and shows its next run.

Before you save, check **What each run covers** in the editor. It shows how many checks run on how many services for each technology, and warns you if a selected check has no selected service to run on.

## Times and time zones

Schedules use UTC, like backups. The Scheduler page shows the time in your time zone next to each UTC time. Because UTC has no daylight saving time, the local time of a run moves by an hour when your clocks change. Days are also picked in UTC, so in your time zone a run can fall on the previous or the next day. The editor tells you when it does.

Each schedule has one time of day. To run checks several times a day, create one schedule for each time.

## Checks and services

**All checks** and **All services** include the checks and services that are added later, such as a new custom check or a newly monitored database. **Selected checks** and **Selected services** cover only what you pick. New checks and services aren't added to them, and the ones that no longer exist are ignored.

If every selected check or every selected service no longer exists, the schedule shows **Targets no existing check** or **Targets no existing service**, and its runs have nothing to do. Edit the schedule to pick other checks or services.

Each run executes checks only on services of their technology. It skips the checks that are disabled in **Advisors > Catalog**, whether for all services or for one. A scheduled run that has nothing to check isn't recorded in **Run history**.

## Email reports

After each run, PMM can email a report of the run's findings. Under **Email report**, choose the recipients:

- **Settings recipients**: The **Default recipients** set under **Configuration > Settings > Advisors**. Changes there apply to the schedule too.
- **These recipients**: Up to 20 email addresses, separated by commas or spaces. Only these addresses get the schedule's reports.
- **Don't email**: PMM sends no report. The results stay in **Insights** and **Run history**.

Unless you choose **Don't email**, also choose the severities under **Severities**:

- **Same as Settings**: The **Default severities** set under **Configuration > Settings > Advisors**. Changes there apply to the schedule too.
- **Only these**: The severities that you select, for this schedule only. You must select at least one of **Critical**, **Error**, **Warning** and **Info**.

A run that has no findings at the chosen severities sends no email. The subject of the email names the schedule.

PMM emails reports only when **Advisor notifications** is turned on under **Configuration > Settings > Advisors**. If it's off, the Scheduler page and the editor show a warning. You can still set recipients in the meantime.

### Send a test email

To check that the reports arrive, click **Send test email** in the editor. PMM sends a sample report to the schedule's recipients, with sample findings for the schedule's severities. The test uses the values in the editor, including the changes that you haven't saved. No checks are run.

**Send test email** is available once the schedule has valid recipients and at least one severity.

## Manage schedules

The **Advisors > Scheduler** page lists the schedules in a table. For each schedule, the table shows when it runs, its checks, services and email report, and its **Next run** and **Last run**. To open the last run in **Run history**, click its result in the **Last run** column.

To pause or resume a schedule, use the **Enabled** switch in its row. A paused schedule keeps its settings but doesn't run.

For the other actions, click the :material-dots-vertical: icon in the schedule's row and select one of the following:

- **Run now**: Starts a run of the schedule right away, with its checks, services and email report. This action is unavailable while another run is queued or in progress.
- **Edit**: Opens the schedule in the editor. To apply your changes, click **Save changes**.
- **Duplicate**: Creates a copy whose name ends with `(copy)`. The copy is paused, so that nothing runs twice. Edit the copy, then enable it.
- **View runs**: Opens **Run history**, filtered by the schedule.
- **Delete**: Deletes the schedule after you confirm. Its future runs are cancelled, and its past runs stay in **Run history** under its name.

### Week view

**This week, UTC** shows the runs of every enabled schedule from Monday to Sunday of the current week, on a UTC time axis. Each schedule has its own color, and a line marks the current time. To see a run's schedule, its time in UTC and in your time zone, and how many checks run on how many services, hover over the run's marker. To open the schedule's details pane, click the marker.

### Details pane

To open the details pane of a schedule, double-click its row, or click one of its markers in the week view. The pane shows the following:

- A summary of when the schedule runs, its checks, services and email report
- **Next runs**, in UTC and in your time zone
- **What each run covers**: Each service in scope, with its technology, environment and cluster, and the number of checks that run on it
- **Checks**: The checks in scope, grouped by technology
- **Recent runs**, each with a link that opens it in **Run history**

To run or edit the schedule from the pane, click **Run now** or **Edit**.

## When schedules run

PMM runs one set of Advisor checks at a time. Schedules follow these rules:

- A schedule that falls due during another run waits, and starts as soon as that run ends.
- Schedules that fall due at the same time run one after another, each as its own run with its own email report.
- A schedule that is already waiting isn't queued again.
- PMM skips the runs that fall due while PMM Server is down, and doesn't catch up on them later. A schedule that is waiting when PMM Server restarts is skipped too.
- PMM starts no run when PMM Server starts or when you turn on Advisors.

In **Run history**, a scheduled run that started late because it waited for another run shows a clock icon. For details, see [Scheduled runs in Run history](advisors.md#scheduled-runs-in-run-history).

If you turn off Advisors, PMM keeps the schedules but doesn't run them, and the Scheduler page shows a banner. When you turn Advisors back on, the schedules run at their next times.

## Default schedule

PMM creates the **All checks daily** schedule when you install or upgrade PMM Server. The schedule runs all checks on all services every day at 06:00 UTC, and emails the default recipients with the default severities.

You can edit, pause or delete **All checks daily** like any other schedule. When you turn on Advisors, PMM creates it again only if no schedule exists.

Without any schedule, Advisor checks run only when someone starts them. To restore the default schedule, click **Create the default schedule** on the empty **Advisors > Scheduler** page.

## Permissions

Users with the **Admin** or **Editor** role can view, create, edit, enable, disable, duplicate, delete and run schedules, and send test emails from the editor. Users with the **Viewer** role can't open the Scheduler.

Because Editors choose the recipients of a schedule, they can send Advisor reports and test emails to any address. The default recipients and severities under **Configuration > Settings > Advisors** stay Admin-only. For the permissions of every role, see [Standard role permissions](../admin/roles/index.md).

## Next steps

- [Review the results of the runs](advisors.md#advisor-results)
- [Develop custom Advisor checks](develop-advisor-checks.md)
