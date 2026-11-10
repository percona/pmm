# Run Advisor checks and review results

Use Advisors to find security risks, misconfigurations and performance problems in your MySQL, PostgreSQL and MongoDB services before they turn into larger issues. 

PMM runs automated checks on your services and lists each finding with the affected service, when it was found and, for most built-in checks, how to fix it.

Checks are grouped by category: Connections, Durability, Logging, Maintenance, Performance, Replication, Resources, Schema and indexes, Security and Versions. 
To browse them, go to **Advisors > Catalog**, or see the [list of advisors and checks](advisor-details.md).

## How Advisor checks run

Advisors are on by default, and PMM runs their checks for you every day: the **All checks daily** schedule checks every service at 06:00 UTC. To check at other times or only some services, [schedule Advisor checks](schedule-advisor-checks.md). To check right away, [run the checks yourself](#run-checks-right-away).

Each run checks only the types of databases that you monitor. For example, if you monitor only PostgreSQL, only the PostgreSQL checks run, even though **Advisors > Catalog** also lists the MySQL and MongoDB checks.

PMM Client runs the checks on your databases, and PMM Server keeps the results. The results never leave PMM Server: PMM doesn't send them as part of Telemetry.

## Get Advisor results by email

PMM can email you a report after each Advisor run that finds issues, so that you learn about them without opening PMM. Only Admins can set this up.

### Set up SMTP

PMM sends the emails through the SMTP server that the `GF_SMTP_*` environment variables of the PMM Server container configure. Set at least `GF_SMTP_ENABLED=true`, `GF_SMTP_HOST` with the server's port, and `GF_SMTP_FROM_ADDRESS`. To set them, [configure the Email (SMTP) server settings](../alert/contact_points.md#configure-email-smtp-server-settings).

### Turn on notifications

To turn on Advisor notifications:
{.power-number}

1. Go to **Configuration > Settings > Advisors**. The notification settings appear only when the **Advisors** option is on.
2. Turn on **Advisor notifications**.
3. In **Default severities**, select the severities of the findings that PMM emails: **Critical**, **Error**, **Warning** or **Info**. You must select at least one. By default, **Critical** and **Error** are selected.
4. In **Default recipients**, enter up to 20 email addresses, separated by commas or spaces. Enter plain addresses, such as `dba@example.com`, without display names.
5. To check that the emails arrive, click **Send test email**. The test uses the values on the page, so you can test them before you apply them. It sends sample findings for the selected severities, runs no checks and records nothing in PMM. If PMM can't send it, for example because a `GF_SMTP_*` variable is missing, an error message shows the cause.
6. Click **Apply changes**.

Each schedule emails these defaults unless it sets its own recipients and severities. To send a schedule's report to other people, [choose who gets the email report](schedule-advisor-checks.md#choose-who-gets-the-email-report). Runs that you start from **Advisors > Catalog**, **Advisors > Insights** or **Advisors > Run history**, such as **Run all** or **Run again**, always use the defaults. **Run now** on a schedule uses that schedule's choices.

### What the email includes

When a run completes with findings at its email severities, PMM sends one email about that run. Runs without such findings send no email, and neither do runs that are interrupted or aborted. For a run of a schedule, the subject names the schedule.

The email lists only the failed checks, not the ones that passed or couldn't run. It contains the following:

- a count of the findings for each selected severity and for each technology: MySQL, PostgreSQL and MongoDB
- suggested next steps
- the details of each finding, in the same format as **Copy as text** on the **Advisors > Insights** page

## Review Advisor results

**Advisors > Insights** lists the result of each check on each service, from every run that PMM keeps.

### Find the issues that need attention

To find the issues that need attention, go to **Advisors > Insights** and do any of the following:

- To see the issues on one service or node, use the **Service** or **Node** filter.
- To see the most urgent issues first, use the **Severity** filter.
- To see one check across all services and runs, click the :material-dots-vertical: icon in its row and select **Filter by name**.
- To see the results of one run only, click the :material-dots-vertical: icon in an insight's row and select **Filter by run ID**.
- To see when PMM found an issue, check the **Checked at** column.

While a run is in progress, the page refreshes every 10 seconds, so you can watch the results come in. For what each status means, see [Insight status](#insight-status).

### Find out how to fix an issue

To find out what to do about an issue, click the :material-dots-vertical: icon in its row and select **View details**. The details pane shows:

- the affected service and node, and when PMM found the issue
- the **Outcome**, which describes the specific problem that the check found
- a **Read more** link to the steps that fix it, when the check provides one

In the list, an icon next to the summary marks the insights that have a **Read more** link.

### Check an issue again after you fix it

To check whether your fix worked without waiting for the next run, click the :material-dots-vertical: icon in the insight's row and select **Re-run now**. PMM runs that check again on the same service.

You can also use **Re-run now** for an insight whose **Status** column shows **Not run**, which means that its run stopped before it reached that check. **Re-run now** isn't available while the **Status** column shows **Pending**: the check is still waiting in the current run, so wait for that run to finish.

### Share insights by email

To send a set of insights to a colleague or another team:
{.power-number}

1. Filter the **Advisors > Insights** page to the insights that you want to share.
2. Click the :material-email-outline: **Email insights** icon in the toolbar.
3. Enter up to 20 email addresses, separated by commas or spaces, and click **Send**.

The email covers every insight that matches the filters, on all pages and with any status, up to 200 insights. If more insights match, the dialog asks you to narrow the filters first. The email lists the filters, a count of the insights with each status, and the details of each insight, in the same format as **Copy as text**.

Sharing insights needs [SMTP](#set-up-smtp), but not **Advisor notifications**, and doesn't change the default recipients.

## Turn off checks that don't apply

Turn off checks that don't apply to your environment, so that their findings don't clutter your results:

- To turn off one check, go to **Advisors > Catalog** and turn off the **Status** switch in the check's row.
- To turn off a check only for some services, click the :material-dns-outline: **Disable for services** icon in the check's row, select the services and click **Disable**. The check keeps running on the other services of its type.
- To stop all Advisor checks, turn off the **Advisors** option under **Configuration > Settings > Advisors**. PMM keeps your [schedules](schedule-advisor-checks.md) but doesn't run them until you turn the option back on.

## Run checks right away

Run checks yourself when you don't want to wait for the next scheduled run, for example to confirm a fix after you change a database setting. Do any of the following:

- To run all checks, go to **Advisors > Catalog** and click the :material-play-outline: **Run all** icon. It's available only when no search or filter is applied.
- To run only some checks, go to **Advisors > Catalog** and find them with **Search checks**, which matches the name, summary, description, category and technology of each check, or with the **Category**, **Technology** and **Status** filters. Then click the :material-playlist-check: **Run selected** icon. PMM runs every enabled check in the list.
- To run one check, go to **Advisors > Catalog** and click the :material-play-outline: **Run** icon in its row.
- To repeat an earlier run with the same checks on the same services, go to **Advisors > Run history**, click the :material-dots-vertical: icon in the run's row and select **Run again**.

To see the results of a run that you started from **Advisors > Catalog**, click **View results** in the confirmation message.

PMM runs one set of checks at a time. If another run is queued or in progress, PMM doesn't start yours, and an error message tells you who started the current run and how long ago. Try again when that run finishes.

## Follow your runs

**Advisors > Run history** lists every run, most recent first, so you can check that your checks ran and what they covered.

### Check the progress of a run

To see where a run is, go to **Advisors > Run history** and check its **Duration**:

- **Queued…**: The run waits for PMM to start it, which takes up to a minute.
- **Running…**: The run is executing checks. The page refreshes every 10 seconds while it runs.
- The run duration: The run finished.
- **Interrupted** or **Aborted**: The run stopped early. To find out why, see [A run shows Interrupted or Aborted](#a-run-shows-interrupted-or-aborted).

### Find out what started a run

Every run is started either by a schedule or by a user, from the PMM UI or the API. Telling them apart helps you check that your schedules ran when you expected, and spot the runs that someone started by hand, for example the run that kept yours from starting.

To find out what started a run, check the **Triggered by** column in **Advisors > Run history**:

- The name of the schedule, such as **All checks daily**, for a run that a schedule started
- **User** and the name of the schedule, such as **User · All checks daily**, for a run that a user started with [**Run now**](schedule-advisor-checks.md#run-pause-or-change-a-schedule) on a schedule
- **User** for any other run that a user started

A run keeps the name that its schedule had when the run started, even if you rename or delete the schedule later.

To see only the runs of one schedule, use the **Schedule** filter. To see only the runs that schedules started, or only the runs that users started, use the **Triggered by** filter.

If a scheduled run started more than a minute late because it waited for another run, a :material-clock-outline: icon appears next to its time in the **Started** column. To see which run it waited for, hover over the icon, for example: *Scheduled for 06:00 UTC. Started 7 min late, after the run started at 05:55 UTC (a user run).* If a schedule started the earlier run, the text names that schedule instead of *a user run*.

### Make sure that a run checked all your services

A run can miss some of your services, for example when the PMM Client that monitors them is disconnected. Before you rely on a run's results, check that it covered everything:
{.power-number}

1. Go to **Advisors > Run history** and find the run.
2. Compare the two numbers in its **Checks** and **Services** columns, shown as *n*/*N*. If *n* is lower than *N* and shows in a warning color, the run didn't complete all the checks or services that it planned. For what the numbers mean, see [Run coverage](#run-coverage). If a service is missing from the run altogether, [make sure that a schedule checks it](schedule-advisor-checks.md#make-sure-that-a-new-database-is-checked).
3. If the **Errors** column shows a number, open the run's insights: double-click the run's row, or click the :material-dots-vertical: icon in its row and select **View insights**.
4. In the **Status** filter, select **Error** to list the checks that couldn't run, and [fix the cause](#an-insight-shows-the-error-status).
5. To run the missed checks again, [repeat the run](#run-checks-right-away).

## Change how long PMM keeps results

PMM keeps runs and their insights for 30 days by default. To keep them for a different period, change **Advisor history retention** under **Configuration > Settings > Advisors**. When a run started longer ago than this period, PMM deletes the run together with all its insights, including **Pending** and **Not run** ones.

## Reference

### Insight status

When a run starts, PMM records a **Pending** insight for every check-and-service pair in the run's plan. As each check finishes, its insight changes to one of the following statuses:

- **OK**: The check passed.
- **Failed**: The check found an issue.
- **Error**: PMM couldn't run the check. The insight shows the reason. To fix it, see [An insight shows the Error status](#an-insight-shows-the-error-status).

If the run ends before it reaches a check, for example because PMM restarted or the HA leader changed, the check's insight changes to **Not run**.

The **Error** status is different from the **Error** severity. The status means that the check couldn't run, while the severity rates an issue that a check found.

**Pending** and **Not run** insights have no severity and no **Checked at** time, so both columns show **—**. The list shows them after the insights of checks that ran, and you can select them in the **Status** filter. To see why a check is pending or didn't run, open the insight's details pane.

### Run coverage

When a run starts, PMM works out every check-and-service pair that the run will execute. This is the run's plan. The **Checks** and **Services** columns in **Run history** show how much of its plan the run covered, as *n*/*N*:

- **Checks**: *n* checks passed or found an issue on at least one service, out of the *N* checks that the run planned.
- **Services**: *n* services had at least one check that passed or found an issue, out of the *N* services that the run planned to cover.

Checks that couldn't run don't count toward *n*. The **Errors** column counts them instead.

Some runs show one of the following values instead:

- **—**: The run has no plan. It is queued or aborted, or it stopped while planning.
- **0/0**: The run completed with nothing to check. There are no MySQL, PostgreSQL or MongoDB services, or every check that applies to them is disabled.

## Troubleshooting

### An insight shows the Error status

If an insight shows the **Error** status, PMM couldn't run the check on that service. To fix it:
{.power-number}

1. Open the insight's details pane and read the reason.
2. If the reason mentions `pmm-agent`, the part of PMM Client that runs the checks, for example that the service has no pmm-agent or that its pmm-agent has never connected, check that PMM Client is installed and connected on the service's node. Then click **Re-run now**.

If the PMM Client that monitors a service is too old for a check, the check doesn't run on that service, so you don't see an insight for it, and it doesn't count toward the run's coverage. To get these checks, upgrade PMM Client on that node.

### A run shows Interrupted or Aborted

If a run in **Run history** shows **Interrupted** or **Aborted** instead of its duration, the run stopped early:

- **Interrupted**: PMM restarted, or the leader of a high availability (HA) cluster changed, before the run finished. PMM keeps the insights that the run saved, and the checks that it didn't reach show as **Not run**. To run them, [run the checks again](#check-an-issue-again-after-you-fix-it) or [repeat the run](#run-checks-right-away).
- **Aborted**: The run stopped before running any check. To find the cause, check the PMM Server logs.

### No email report arrives

If a run finds issues but no report arrives, see [No email report arrives](schedule-advisor-checks.md#no-email-report-arrives).

## Related topics

- [Schedule Advisor checks](schedule-advisor-checks.md)
- [List of advisors and checks](advisor-details.md)
- [Develop Advisor checks](develop-advisor-checks.md)
