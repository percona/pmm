# Installing the PMM Client Package on Red Hat and CentOS

If you are running an RPM-based Linux distribution, use the **yum** package
manager to install PMM Client from the official Percona software repository.

Percona provides `.rpm` packages for 64-bit versions
of Red Hat Enterprise Linux 6 (Santiago) and 7 (Maipo),
including its derivatives that claim full binary compatibility,
such as, CentOS, Oracle Linux, Amazon Linux AMI, and so on.

**NOTE**: PMM Client should work on other RPM-based distributions,
but it is tested only on RHEL and CentOS versions 6 and 7.

To install the PMM Client package, complete the following procedure. Run the following commands as root or by using the **sudo** command:


1. Configure Percona repositories as described in
[Percona Software Repositories Documentation](https://www.percona.com/doc/percona-repo-config/index.html).


2. Install the `pmm-client` package:

```
yum install pmm-client
```

**NOTE**: You can also download PMM Client packages from the [PMM download page](https://www.percona.com/downloads/pmm/).
Choose the appropriate PMM version and your GNU/Linux distribution in
two pop-up menus to get the download link (e.g. *Percona Monitoring and Management 1.17.2* and *Red Hat Enterprise Linux / CentOS / Oracle Linux 7*).

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
