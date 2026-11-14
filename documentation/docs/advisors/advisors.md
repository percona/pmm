# Run Advisor checks and review results

Use Advisors to find security risks, misconfigurations and performance problems in your MySQL, PostgreSQL and MongoDB services before they turn into larger issues.

PMM runs automated checks on your services and lists each finding with the affected service, when it was found and, for most built-in checks, how to fix it.

To see which checks PMM runs, go to **Advisors > Catalog**, or see the [list of advisors and checks](advisor-details.md). To find the checks for one area, such as **Security** or **Replication**, use the **Category** filter in the catalog.

## How Advisor checks run

Advisors are on by default. Every day at 06:00 UTC, the **All checks daily** schedule runs every check on every service. To run checks at other times or on only some services, [schedule them](schedule-advisor-checks.md). To run them right away, [start a run yourself](#run-checks-right-away).

**Advisors > Catalog** lists the checks for all supported databases, but you get results only for the databases that you monitor. For example, if you monitor only PostgreSQL, PMM runs only the PostgreSQL checks and skips the MySQL and MongoDB ones.

PMM Client runs these checks on your databases, and PMM Server keeps the results. The results never leave PMM Server: PMM doesn't send them as part of Telemetry.

PMM runs one set of checks at a time, whether a schedule or a user starts it. A run that you start while another one is queued or in progress doesn't start.

## Get Advisor results by email

To learn about issues without opening PMM, have PMM email you a report after each Advisor run that finds issues. Only Admins can turn this on.

PMM sends these emails through the SMTP server of PMM Server. If you haven't set one up yet, [configure the Email (SMTP) server settings](../alert/contact_points.md#configure-email-smtp-server-settings) first.

### Turn on notifications

To turn on Advisor notifications:
{.power-number}

1. Go to **Configuration > Settings > Advisors**. The notification settings appear only when the **Advisors** option is on.
2. Turn on **Advisor notifications**.
3. In **Default severities**, select the severities of the findings that PMM emails: **Critical**, **Error**, **Warning** or **Info**. Select at least one. By default, **Critical** and **Error** are selected.
4. In **Default recipients**, enter up to 20 email addresses, separated by commas or spaces. Enter plain addresses, such as `dba@example.com`, without display names.
5. To check that the emails arrive, click **Send test email**. The test uses the values on the page, so you can test them before you apply them. It sends sample findings for the selected severities, runs no checks and records nothing in PMM. If the test fails, [find out why the test email fails](#the-test-email-fails).
6. Click **Apply changes**.

### What the email includes

PMM sends one email after each run that finds issues at the selected severities. If a run finds no issues at those severities, or is interrupted or aborted, PMM sends no email.

Who gets the email, and which severities it reports, depends on where the run started:

- Runs started from a schedule send the email to the schedule's recipients, with the schedule's severities. These are the [default recipients and severities](#turn-on-notifications), unless you [set custom recipients and severities for the schedule](schedule-advisor-checks.md#choose-who-gets-the-email-report). The subject of the email includes the schedule's name, for example `PMM Advisor Insights: 3 finding(s) for All checks daily`.
- Runs started manually from **Advisors > Catalog**, **Advisors > Insights** or **Advisors > Run history** always send the email to the default recipients, with the default severities. The subject of the email includes the run's ID, for example `PMM Advisor Insights: 3 finding(s) for run 6f1c2b9e-4d3a-4b8e-9c71-2a5e8f0d1b37`.

The email covers only the failed checks, not the ones that passed or couldn't run, and includes:

- a count of the findings for each selected severity and for each technology: MySQL, PostgreSQL and MongoDB
- suggested next steps
- the details of each finding, in the same format as **Copy as text** on the **Advisors > Insights** page

## Review Advisor results

To see what the checks found on your services, go to **Advisors > Insights**. You can see the results of every run from the last 30 days, or for as long as you [set PMM to keep results](#change-how-long-pmm-keeps-results). Each result has a [status](#understand-the-status-of-an-insight) that tells you whether the check passed, found an issue or couldn't run.

### Find the issues that need attention

To narrow the list down to the issues that need your attention:

- filter by **Severity** to start with the most urgent issues
- filter by **Service** or **Node** to focus on the services and nodes that you look after
- click the :material-dots-vertical: icon in an issue's row and select **Filter by name** to find out whether the issue also affects other services
- click the :material-dots-vertical: icon in an insight's row and select **Filter by run ID** to review what a single run found, such as a run that you just started
### Understand the status of an insight

The status of an insight tells you whether its check passed, found an issue, couldn't run, or hasn't run yet.

When a run starts, PMM records a **Pending** insight for every check-and-service pair in the run's plan. As each check finishes, its insight changes to one of these statuses:

- **OK**: The check passed.
- **Failed**: The check found an issue.
- **Error**: PMM couldn't run the check. The insight shows the reason. [Fix the cause of the error](#an-insight-shows-the-error-status).

If the run ends before it reaches a check, for example because PMM restarted or the HA leader changed, the check's insight changes to **Not run**.

The **Error** status is different from the **Error** severity. The status means that the check couldn't run, while the severity rates an issue that a check found.

**Pending** and **Not run** insights have no severity and no **Checked at** time, so both columns show **—**. The list shows them after the insights of checks that ran, and you can select them in the **Status** filter. To see why a check is pending or didn't run, open the insight's details pane.

### Fix an issue

To fix an issue and check that the fix worked:
{.power-number}

1. Click the :material-dots-vertical: icon in the issue's row and select **View details**.
2. Read the **Outcome** to learn what exactly the check found on the affected service, and when.
3. Click **Read more** and follow the steps that fix the issue. Not every check has these steps: in the list, an icon next to the summary marks the issues that do.
4. To check your fix without waiting for the next run, click the :material-dots-vertical: icon in the issue's row and select **Re-run now**. PMM runs the check again on the same service. If the issue is still **Pending** in a run in progress, wait for that run to finish first.

### Share insights by email

You can send a set of insights to a colleague or another team, for example the owners of the affected services. You don't need **Advisor notifications** for this, but PMM Server needs to be able to [send email through SMTP](../alert/contact_points.md#configure-email-smtp-server-settings).

To share insights by email:
{.power-number}

1. Filter the **Advisors > Insights** page to the insights that you want to share. The email includes every insight that matches the filters, not only the ones on the current page, up to 200. If more insights match, narrow the filters further.
2. Click the :material-email-outline: **Email insights** icon in the toolbar.
3. Enter up to 20 email addresses, separated by commas or spaces, and click **Send**. The default recipients don't change.

The email lists the filters that you used, a count of the insights with each status, and the details of each insight, in the same format as **Copy as text**.

## Turn off checks that don't apply

Some findings are correct but don't matter in your environment, for example when a check flags a setting that you chose on purpose, or recommends a production setting for a test server. Turn off these checks, so that their findings don't clutter your results. In **Advisors > Catalog**:

- turn off the **Status** switch in a check's row to stop the check on all services
- click the :material-dns-outline: **Disable for services** icon in a check's row, select the services and click **Disable** to stop the check only on those services, while it keeps running on the others

If you don't want any Advisor checks to run, turn off the **Advisors** option under **Configuration > Settings > Advisors**. PMM keeps your [schedules](schedule-advisor-checks.md) but doesn't run them until you turn the option back on.

## Run checks right away

Run checks yourself when you don't want to wait for the next scheduled run, for example to confirm a fix after you change a database setting.

To run checks right away:
{.power-number}

1. Go to **Advisors > Catalog** and start the run:

    - click the :material-play-outline: **Run** icon in a check's row to run only that check
    - find checks by name, summary, description, category or technology with **Search checks**, or with the **Category**, **Technology** and **Status** filters, then click the :material-playlist-check: **Run selected** icon to run every enabled check in the list
    - clear any search and filters, then click the :material-play-outline: **Run all** icon to run every check

    If an error message says that another run is in progress, wait for that run to finish and try again. The message tells you who started it and how long ago.

2. In the confirmation message, click **View results**. While the run is in progress, **Advisors > Insights** refreshes every 10 seconds, so you can watch the results come in.

## Follow your runs

Before you rely on the results of a run, or when results or an email report that you expected don't show up, check in **Advisors > Run history** that the run started when you expected, finished, and reached all your services.

### Check the progress of a run

To find out whether a run has finished, for example before you review its results, go to **Advisors > Run history** and check its **Duration**:

- **Queued…**: The run waits for PMM to start it, which takes up to a minute.
- **Running…**: The run is executing checks. The page refreshes every 10 seconds while it runs.
- The run duration: The run finished.
- **Interrupted** or **Aborted**: The run stopped early. [Find out why the run stopped](#a-run-shows-interrupted-or-aborted).

### Find out what started a run

When a run surprises you, for example because it ran at an unexpected time or kept your own run from starting, find out whether a schedule or a user started it. A user can start a run from the PMM UI or the API. To find out, check the **Triggered by** column in **Advisors > Run history**, which shows:

- the schedule's name, such as **All checks daily**, if a schedule started the run
- **User** and the schedule's name, such as **User · All checks daily**, if a user started the schedule with [**Run now**](schedule-advisor-checks.md#run-pause-or-change-a-schedule)
- **User** if a user started the run in any other way

A run keeps the name that its schedule had when the run started, even if you rename or delete the schedule later.

To check that a schedule ran when you expected, filter by **Schedule**. To tell scheduled runs from the runs that users started, filter by **Triggered by**.

If a scheduled run started more than a minute late because it waited for another run to finish, a :material-clock-outline: icon appears next to its time in the **Started** column. Hover over the icon to see:

- when the run was scheduled and how late it started, for example *Scheduled for 06:00 UTC. Started 7 min late*
- when the earlier run that it waited for started, and who started it, for example *after the run started at 05:55 UTC (a user run)*. If a schedule started the earlier run, the text names that schedule instead of *a user run*.

### Make sure that a run checked all your services

A run can miss some of your services, for example when the PMM Client that monitors them is disconnected. Before you rely on a run's results, check that it covered everything:
{.power-number}

1. Go to **Advisors > Run history** and find the run.
2. Compare the two numbers in its **Checks** and **Services** columns, shown as *n*/*N*. If *n* is lower than *N* and shows in a warning color, the run didn't complete all the checks or services that it planned. Learn [what the coverage numbers mean](#understand-the-coverage-numbers-of-a-run). If a service is missing from the run altogether, [make sure that a schedule checks it](schedule-advisor-checks.md#make-sure-that-a-new-database-is-checked).
3. If the **Errors** column shows a number, open the run's insights: double-click the run's row, or click the :material-dots-vertical: icon in its row and select **View insights**.
4. In the **Status** filter, select **Error** to list the checks that couldn't run, and [fix the cause](#an-insight-shows-the-error-status).
5. After you fix the cause, click the :material-dots-vertical: icon in the run's row and select **Run again**. PMM runs the same checks on the same services again.

### Understand the coverage numbers of a run

The **Checks** and **Services** columns in **Run history** tell you whether a run reached everything that it planned to check.

When a run starts, PMM works out every check-and-service pair that the run will execute. This is the run's plan. The two columns show how much of its plan the run covered, as *n*/*N*:

- **Checks**: *n* checks passed or found an issue on at least one service, out of the *N* checks that the run planned.
- **Services**: *n* services had at least one check that passed or found an issue, out of the *N* services that the run planned to cover.

Checks that couldn't run don't count toward *n*. The **Errors** column counts them instead.

Some runs show one of these values instead:

- **—**: The run has no plan. It is queued or aborted, or it stopped while planning.
- **0/0**: The run completed with nothing to check. There are no MySQL, PostgreSQL or MongoDB services, or every check that applies to them is disabled.

## Change how long PMM keeps results

Keep results longer to compare runs over a longer period, or for a shorter time to use less storage on PMM Server. By default, PMM keeps runs and their insights for 30 days.

To change this, set **Advisor history retention** under **Configuration > Settings > Advisors**. PMM deletes a run, with all its insights, once the run started longer ago than this period.

## Troubleshooting

Fix the most common problems with Advisor checks: checks that can't run, runs that stop early, and email that doesn't arrive.

### An insight shows the Error status

If an insight shows the **Error** status, PMM couldn't run the check on that service. To fix it:
{.power-number}

1. Open the insight's details pane and read the reason.
2. If the reason mentions `pmm-agent`, which is part of PMM Client, check that PMM Client is installed and connected on the service's node.
3. Click **Re-run now** to run the check again.

If the PMM Client that monitors a service is too old for a check, the check doesn't run on that service, so you don't see an insight for it, and it doesn't count toward the run's coverage. To get these checks, [upgrade PMM Client](../pmm-upgrade/upgrade_client.md) on that node.

### A run shows Interrupted or Aborted

If a run in **Run history** shows **Interrupted** or **Aborted** instead of its duration, the run stopped early:

- **Interrupted**: PMM restarted, or the leader of a high availability (HA) cluster changed, before the run finished. PMM keeps the insights that the run saved, and the checks that it didn't reach show as **Not run**. To run one of them, click the :material-dots-vertical: icon in its row on **Advisors > Insights** and select **Re-run now**. To run them all, go to **Advisors > Run history**, click the :material-dots-vertical: icon in the run's row and select **Run again**.
- **Aborted**: The run stopped before running any check. To find the cause, check the PMM Server logs.

### The test email fails

If **Send test email** shows an error, check the message:

- If it starts with **SMTP is not configured in PMM Server**, or mentions **GF_SMTP_HOST** or the SMTP port, it names the `GF_SMTP_*` variable to fix. Set the variable as described in [Configure Email (SMTP) server settings](../alert/contact_points.md#configure-email-smtp-server-settings), then recreate the PMM Server container with the same data volume. Restarting the container isn't enough, because a restart keeps the old environment variables.
- Any other error comes from your SMTP server. Check the host, port, user name and password with your email administrator.

### No email report arrives

If a run finds issues but no report arrives, [find out why the email report doesn't arrive](schedule-advisor-checks.md#no-email-report-arrives).

## Related topics

- [Schedule Advisor checks](schedule-advisor-checks.md)
- [List of advisors and checks](advisor-details.md)
- [Develop Advisor checks](develop-advisor-checks.md)
