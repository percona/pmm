# Installing PMM Client on Debian or Ubuntu

If you are running a DEB-based Linux distribution, use the **apt** package
manager to install PMM Client from the official Percona software repository.

Percona provides `.deb` packages for 64-bit versions of the following
distributions:


* Debian 8 (jessie)


* Debian 9 (stretch)


* Ubuntu 14.04 LTS (Trusty Tahr)


* Ubuntu 16.04 LTS (Xenial Xerus)


* Ubuntu 16.10 (Yakkety Yak)


* Ubuntu 17.10 (Artful Aardvark)


* Ubuntu 18.04 (Bionic Beaver)

**NOTE**: PMM Client should work on other DEB-based distributions, but it is tested
only on the platforms listed above.

To install the PMM Client package, complete the following
procedure. Run the following commands as root or by using the **sudo** command:


1. Configure Percona repositories as described in [Percona Software
Repositories Documentation](https://www.percona.com/doc/percona-repo-config/index.html).


2. Install the PMM Client package:

```
$ apt-get install pmm-client
```

**NOTE**: You can also download PMM Client packages from the [PMM download page](https://www.percona.com/downloads/pmm/).
Choose the appropriate PMM version and your GNU/Linux distribution in
two pop-up menus to get the download link (e.g. *Percona Monitoring and Management 1.17.2* and *Ubuntu 18.04 (Bionic Beaver*).

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
