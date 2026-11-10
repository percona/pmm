# Schedule Advisor checks

Use Advisor schedules to run checks automatically at times that suit your team, on the services that matter to you, and to email the results to the people who act on them.

When you install or upgrade PMM Server, PMM creates one schedule, **All checks daily**, which runs every check on every service each day at 06:00 UTC and emails the [default recipients](advisors.md#get-advisor-results-by-email). You can keep it, change it, or replace it with your own schedules.

## Prerequisites

Before you create a schedule:
{.power-number}

1. Check that the **Advisors** option is turned on under **Configuration > Settings > Advisors**. Schedules run only while Advisors are on.
2. Check that you have the [**Editor** or **Admin** role](../admin/roles/index.md). Viewers can't open the Scheduler. Editors can send Advisor reports to any email address, so give the Editor role only to users you trust with your Advisor findings.
3. To get the email reports, go to **Configuration > Settings > Advisors**, check that **Advisor notifications** is on, and click **Send test email**. If the switch is off or the test email doesn't arrive, [set up email notifications](advisors.md#get-advisor-results-by-email).

## Create a schedule

By default, the **All checks daily** schedule runs every check on every service at 06:00 UTC. Create your own schedule when that doesn't fit your team, for example to check production services more often, or to send security findings to your security team.

To create a schedule:
{.power-number}

1. Go to **Advisors > Scheduler** and click **New schedule**.
2. In **Name**, enter a name that no other schedule uses, up to 100 characters. The name appears in **Run history** and in the subject of the email report, so choose one that your team recognizes.
3. Under **When**, select the **Days** and enter the **Time (UTC)** as `HH:MM`. To pick several days at once, click **every day**, **weekdays** or **weekends** next to **Set to**.

    To see when the schedule runs in your local time, check the preview under **When**. It lists the next runs in UTC and in your time zone. Because schedules use UTC, a run can fall on a different day in your time zone, and the preview tells you when it does. The local time also moves by an hour when your clocks change for daylight saving time.

4. Under **Checks**, choose which checks run:

    - **All checks**: every check, including the ones added later, such as a new custom check
    - **Selected checks**: only the checks that you pick. To find them, use **Search checks** and the **Category** filter.

5. Under **Services**, choose which services the checks run on:

    - **All services**: every service, including the ones you add later
    - **Selected services**: only the services that you pick. To find them, use the search field and the **Environment** filter.

6. Under **What each run covers**, check how many checks run on how many services for each technology. If some selected checks have no service of their technology to run on, or some selected services get no checks, a message names them. Change your selection if that's not what you want.
7. Under **Email report**, [choose who gets the email report](#choose-who-gets-the-email-report).
8. Click **Create schedule**.

### Run checks several times a day

Each schedule runs at one time of day. To run checks several times a day, create one schedule for each time. The quickest way is to duplicate a schedule:
{.power-number}

1. In the schedule's row, click the :material-dots-vertical: icon and select **Duplicate**. The copy's name ends with `(copy)`, and the copy is paused, so that nothing runs twice.
2. In the copy's row, click the :material-dots-vertical: icon and select **Edit**. Change the name and the **Time (UTC)**, then click **Save changes**.
3. Turn on the copy's **Enabled** switch.

## Choose who gets the email report

After each run, a schedule emails a report of the issues it found. You can choose, for each schedule, who gets that report and which severities it includes.

### Check the default recipients and severities

Every schedule emails the default recipients, with the default severities, until you change them. Before you change a schedule's report, check what these defaults are:
{.power-number}

1. Open the schedule in the editor: click the :material-dots-vertical: icon in its row and select **Edit**.
2. Under **Email report**, check **Settings recipients**. It shows the default recipients from **Configuration > Settings > Advisors**, or says that none are set. If the defaults change there, the schedule uses the new recipients.
3. Under **Severities**, check **Same as Settings**. It shows the default severities, which are **Critical** and **Error** unless an Admin changed them. If the defaults change, the schedule uses the new severities.

Only Admins can [change the defaults](advisors.md#turn-on-notifications).

### Send a schedule's report to other recipients

To send a schedule's report to other people, for example the team that looks after the services it checks, replace the default recipients with your own list. The default recipients then no longer get this schedule's report, so if they should still get it, add their addresses to your list too:
{.power-number}

1. Open the schedule in the editor: click the :material-dots-vertical: icon in its row and select **Edit**, or [create a new schedule](#create-a-schedule).
2. Under **Email report**, select **These recipients** and enter up to 20 email addresses, separated by commas or spaces. Only these people get this schedule's report, and its subject names the schedule.
3. To check that the report arrives, click **Send test email**. PMM sends a sample report with sample findings and runs no checks. The test uses the values in the editor, so you can test before you save. The button is available once the email addresses are valid and at least one severity is selected.
4. Click **Save changes**, or **Create schedule** for a new schedule.

### Change which severities a schedule reports

By default, the email report lists the findings at the [default severities](#check-the-default-recipients-and-severities). To report a different set for one schedule, for example to add **Warning** for production services, or to report only **Critical** for a test environment:
{.power-number}

1. Open the schedule in the editor.
2. Under **Severities**, select **Only these**, then select at least one of **Critical**, **Error**, **Warning** and **Info**.
3. Click **Save changes**.

PMM sends a report only when a run finds issues at these severities.

### Stop a schedule from emailing

To keep running a schedule without emailing anyone, for example while you test it:
{.power-number}

1. Open the schedule in the editor.
2. Under **Email report**, select **Don't email**. The results still appear on the **Insights** and **Run history** pages.
3. Click **Save changes**.

To go back to the defaults later, select **Settings recipients** and **Same as Settings**.

## Check what your schedules will run

Check your schedules when something changes in your environment. For example, before planned maintenance, make sure that no run is due during it. After you add a database, make sure that a schedule checks it. When you add a schedule, make sure that its runs don't collide with the runs of other schedules.

### Keep runs out of a maintenance window

Advisor checks query your databases, so plan runs outside your maintenance windows. To find the runs that fall in a window and move them:
{.power-number}

1. Go to **Advisors > Scheduler** and find the runs in **This week, UTC**.
2. To see which schedule a run belongs to, and its time in your time zone, hover over the run's marker.
3. If the run falls in the window, [change the schedule's time](#run-pause-or-change-a-schedule), or turn off its **Enabled** switch until the maintenance is over.

### Make sure that a new database is checked

Schedules set to **All services** check a new database automatically. Schedules set to **Selected services** check only the services that you picked, so a new database isn't included. To find out which schedules check a new database:
{.power-number}

1. Go to **Advisors > Scheduler** and open a schedule's details pane: double-click its row, or click one of its runs in **This week, UTC**.
2. Under **What each run covers**, find the database's service and the number of checks that run on it. If the service is missing, the schedule doesn't check it.
3. To see which checks run, look under **Checks**. They're grouped by technology.
4. If no schedule checks the database, [edit a schedule](#run-pause-or-change-a-schedule) and add the service, or set the schedule to **All services**.

### Spread out runs that fall close together

PMM runs one set of Advisor checks at a time. When two schedules fall due close together, the second one waits until the first run finishes, and starts late. Schedules due at the same time run one after another, each with its own email report. To keep runs on time:
{.power-number}

1. Go to **Advisors > Scheduler** and look for runs that sit close together in **This week, UTC**.
2. To see which schedules they belong to, hover over the runs' markers.
3. To find out how long a run takes, [check its **Duration**](advisors.md#check-the-progress-of-a-run) in **Advisors > Run history**.
4. [Change the time](#run-pause-or-change-a-schedule) of one schedule so that its runs start after the other schedule's runs finish.

## Run, pause or change a schedule

Adjust your schedules as your environment changes. Go to **Advisors > Scheduler**, then:

- To check whether a fix worked without waiting for the next run, click the :material-dots-vertical: icon in the schedule's row and select **Run now**. The run uses the schedule's checks, services and email report. You can't start it while another run is queued or in progress.
- To stop a schedule during an outage or a migration without losing its settings, turn off the **Enabled** switch in its row. Turn it back on when you're done.
- To run checks at another time or on other services, click the :material-dots-vertical: icon in the schedule's row, select **Edit**, make your changes and click **Save changes**.
- To see what a schedule found in earlier runs, click the :material-dots-vertical: icon in its row and select **View runs**. To open only its last run, click the result in the **Last run** column. To learn what **Run history** shows, see [Follow your runs](advisors.md#follow-your-runs).
- To remove a schedule that you no longer need, click the :material-dots-vertical: icon in its row, select **Delete** and confirm. Its past runs stay in **Run history** under its name.

## Restore the default schedule

**All checks daily** is an ordinary schedule, so you can edit, pause or delete it like any other. If you delete it and all your other schedules, Advisor checks run only when someone starts them.

To get the daily run back, go to the empty **Advisors > Scheduler** page and click **Create the default schedule**. PMM also creates it again when you turn on Advisors and no schedule exists.

## Troubleshooting

### A scheduled run started late or didn't run

If a scheduled run started later than its time, or didn't run at all, find out why:
{.power-number}

1. Check whether another run was in progress. In **Run history**, a :material-clock-outline: icon next to the run's start time means that it waited for another run. To [find out which one](advisors.md#find-out-what-started-a-run), hover over the icon. To avoid the wait, [spread out the runs](#spread-out-runs-that-fall-close-together).
2. Check that the schedule's **Enabled** switch is on.
3. Check that Advisors are on. If they're off, the Scheduler page shows a banner and no schedule runs. When you turn them back on, the schedules run at their next times. PMM doesn't start a run right away.
4. Check whether PMM Server was down or restarted when the run was due. PMM skips the runs that fall due while it's down, including a run that was waiting when it restarted, and doesn't catch up on them later.
5. Check whether the run had anything to check. A scheduled run with nothing to check isn't recorded in **Run history**. This happens when all its checks are [turned off](advisors.md#turn-off-checks-that-dont-apply) in **Advisors > Catalog**, for all services or for its services, or when its services don't match the technology of its checks.

### A schedule shows Targets no existing check or Targets no existing service

If a schedule shows **Targets no existing check** or **Targets no existing service**, every check or service that you selected for it no longer exists, so its runs have nothing to do. **Selected checks** and **Selected services** cover only what you picked, and don't add new checks or services. If only some of them no longer exist, the schedule skips those and runs the rest.

To fix it, edit the schedule and select other checks or services, or switch to **All checks** or **All services**.

### No email report arrives

If a schedule runs but no report arrives:
{.power-number}

1. Check that the run found issues at the schedule's severities. A run without such findings sends no email.
2. Check that **Advisor notifications** is on under **Configuration > Settings > Advisors**. If it's off, the Scheduler page and the editor show a warning, and no schedule sends email. The recipients that you set in the meantime are kept.
3. In the schedule editor, click **Send test email**. If the test email doesn't arrive either, [check the SMTP settings of PMM Server](advisors.md#set-up-smtp).

## Related topics

- [Review the results of the runs](advisors.md#review-advisor-results)
- [Turn on email notifications](advisors.md#get-advisor-results-by-email)
- [Develop Advisor checks](develop-advisor-checks.md)
