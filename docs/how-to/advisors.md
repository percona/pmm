# Work with Advisor checks

Advisors are automated checks that you can run against connected databases to identify any potential security threats, configuration problems, performance concerns, policy non-compliance issues etc. 

Checks are grouped into advisors according to the functionality and recommendations they provide.

## Prerequisites for accessing Advisor checks
All checks are hosted on Percona Platform. PMM Server automatically downloads them from here when the **Advisors** and **Telemetry** options are enabled in PMM under **Configuration > Settings > Advanced Settings**. Both these options are enabled by default.

### Advisor check tiers and Platform entitlements
Depending on the entitlements available for your Percona Account, the set of advisor checks that PMM can download from the Percona Platform differs in terms of complexity and functionality. 

If your PMM instance is not connected to Percona Platform, PMM can only download the basic set of Anonymous Advisor checks. 
As soon as you connect your PMM instance to Percona Platform, has access to additional checks, available only for Registered PMM instances. 

If you are a Percona customer with a Percona Customer Portal account, you also get access to Paid Advisor checks, which offer more advanced database health information.

​To see the complete list of available checks, see the [Advisor Checks for PMM](https://docs.percona.com/percona-platform/checks.html) topic in the Percona Platform documentation.  

## Enable/Disable
To download the checks available for your Percona Account, the Advisors and Telemetry options have to be enabled under <i class="uil uil-cog"></i> **Configuration <i class="uil uil-setting"></i> > Settings > Advanced Settings**.

These options are enabled by default so that PMM can run automatic advisor checks in the background. However, you can disable them at any time if you do not need to check the health and performance of your connected databases.

## Automatic checks
Advisor checks can be executed manually or automatically. 
By default, PMM runs all the checks available for your PMM instances every 24 hours. 
### Change run interval for automatic advisors
 You can change the standard 24-hours interval to a custom frequency for each advisor:

 - *Rare interval*   -  78 hours       
 - *Standard interval* (default) -  24 hours
 - *Frequent interval*   - 4   hours

To change the frequency of an automatic advisor:

1. Click **{{icon.checks}} Advisors**.
2. Select the **All** tab.
3. In the **Actions** column for a chosen check, click the ![Edit](..//_images/edit.png)  **Interval** icon.
4. Chose an interval and click **Save**.

## Manual checks
In addition to the automatic checks that run every 24 hours, you can also run checks manually, for ad-hoc assessments of your database health and performance.

To manually run all checks or individual ones:
 
1. Click **{{icon.checks}} Advisors** on the main menu.
2. Select the **All** tab.
3. Click **Run checks** to run all the available advisors at once, or click **Run** next to each check to run a single check individually.
![!Actions options](../_images/PMM_Checks_Actions.png)

## Checks results
The results are sent to PMM Server where you can review any failed checks on the **Home Dashboard > Failed Advisors Checks** panel. The summary count of failed checks is classified as <b style="color:#e02f44;">Critical</b>, <b style="color:#e36526;">Major</b> and <b style="color:#5794f2;">Trivial</b>:

![!Failed Advisors Checks panel](../_images/PMM_Home_Dashboard_Panels_Failed_Checks.jpg)

To see more details about the available checks and any checks that failed, click the *{{icon.checks}} Advisors* icon on the main menu. 

**Check results data *always* remains on the PMM Server.** This is not related to anonymous data sent for Telemetry purposes.
