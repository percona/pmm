# Security Threat Tool

The Security Threat Tool runs regular checks against connected databases, alerting you if any servers pose a potential security threat.

### Anonymous and registered checks
All checks are hosted on Percona Platform. PMM Server automatically downloads them from here when the Security Threat Tool is enabled in PMM. 

By default, PMM has access to a set of anonymous checks, which can be downloaded even if PMM is not connected to Percona Platform. 
As soon as you connect your PMM instance to Percona Platform, you get additional access to registered checks, which offer more advanced database health information.

​To see the complete list of available checks, see the [Security Checks for PMM](https://docs.percona.com/percona-platform/checks.html) topic in the Percona Platform documentation.  


## How to enable

By default, the Security Threat Tool (STT) is disabled. To enable it, select <i class="uil uil-cog"></i> *Configuration* → <i class="uil uil-setting"></i> *Settings* → *Advanced Settings*. ([Read more](../how-to/configure.md#advanced-settings)).

Enabling STT in the settings causes the PMM server to download STT checks from Percona Platform and run them once. This operation runs in the background, so even though the settings update finishes instantly, it might take some time for the checks to complete download and execution and the results (if any) to be visible in the *PMM Database Checks* dashboard.

## Checks results
The results are sent to PMM Server where you can review any failed checks on the **Home Dashboard > Failed security checks** panel. The summary count of failed checks is classified as <b style="color:#e02f44;">Critical</b>, <b style="color:#e36526;">Major</b> and <b style="color:#5794f2;">Trivial</b>:

![!Failed security checks panel](../_images/PMM_Home_Dashboard_Panels_Failed_Security_Checks.jpg)

To see more details about the available checks and any checks that failed, click the *{{icon.checks}} Security Checks* on the main menu. This icon is only available if you have enabled the Security Threat Tool.

**Check results data *always* remains on the PMM Server.** It is not related to anonymous data sent for Telemetry purposes.

## Change a check's interval
The checks can be executed manually or automatically. By default, PMM runs automatic checks every 24 hours. To configure this interval:


1. Click *{{icon.checks}} Security Checks*.

2. Select the *All Checks* tab.

3. In the *Actions* column for a chosen check, click the <i class="uil uil-history"></i> *Interval* icon.

4. Chose an interval: *Standard*, *Rare*, *Frequent*.

5. Click *Save*.
