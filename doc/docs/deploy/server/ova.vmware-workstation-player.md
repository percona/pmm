# VMware Workstation Player

The following procedure describes how to run the *PMM Server* appliance
using VMware Workstation Player:


1. Download the OVA. The latest version is available at the [Download Percona Monitoring and Management](https://www.percona.com/downloads/pmm) site.


2. Import the appliance.


    1. Open the File menu and click Open.


    2. Specify the path to the OVA and click Continue.

**NOTE**: You may get an error indicating that import failed.
Simply click Retry and import should succeed.


3. Configure network settings to make the appliance accessible
from other hosts in your network.

If you are running the applianoce on a host
with properly configured network settings,
select **Bridged** in the **Network connection** section
of the appliance settings.


4. Start the PMM Server appliance and set the root password (required on the first login)

If it was assigned an IP address on the network by DHCP,
the URL for accessing PMM will be printed in the console window.


5. Set the root password as described in the section

<!-- -*- mode: rst -*- -->
<!-- Tips (tip) -->
<!-- Abbreviations (abbr) -->
<!-- Docker commands (docker) -->
<!-- Graphical interface elements (gui) -->
<!-- Options and parameters (opt) -->
<!-- pmm-admin commands (pmm-admin) -->
<!-- SQL commands (sql) -->
<!-- PMM Dashboards (dbd) -->
<!-- * Text labels -->
<!-- Special headings (h) -->
<!-- Status labels (status) -->
