# Node Exporter (node_exporter)

The following options may be passed to the `linux:metrics` monitoring service as additional options. For more information about this exporter see its GitHub repository: [https://github.com/percona/node_exporter](https://github.com/percona/node_exporter).

## Collector options

| Name         | Enabled by Default | Description |
| ------------ | ------------------ | ------------------------------------------------------------------------------------ |
| conntrack    | Yes                | Shows conntrack statistics (does nothing if no /proc/sys/net/netfilter/ present). |
| diskstats    | Yes                | Disk I/O statistics from /proc/diskstats. |
| edac         | Yes                | Error detection and correction statistics. |
| entropy      | Yes                | Available entropy. |
| filefd       | Yes                | File descriptor statistics from /proc/sys/fs/file-nr. |
| filesystem   | Yes                | Filesystem statistics, such as disk space used. , Dragonfly, FreeBSD, Linux, OpenBSD |
| hwmon        | Yes                | Hardware monitoring and sensor data from /sys/class/hwmon/. |
| infiniband   | Yes                | Network statistics specific to InfiniBand configurations. |
| loadavg      | Yes                | Load average. , Dragonfly, FreeBSD, Linux, NetBSD, OpenBSD, Solaris |
| mdadm        | Yes                | Statistics about devices in /proc/mdstat (does nothing if no /proc/mdstat present). |
| meminfo      | Yes                | Memory statistics. , Dragonfly, FreeBSD, Linux |
| netdev       | Yes                | Network interface statistics such as bytes transferred. , Dragonfly, FreeBSD, Linux, OpenBSD |
| netstat      | Yes                | Network statistics from /proc/net/netstat. This is the same information as netstat -s. |
| sockstat     | Yes                | Various statistics from /proc/net/sockstat. |
| stat         | Yes                | Various statistics from /proc/stat. This includes CPU usage, boot time, forks and interrupts. |
| textfile     | Yes                | Statistics read from local disk. The –collector.textfile.directory flag must be set. |
| time         | Yes                | The current system time. |
| uname        | Yes                | System information as provided by the uname system call. |
| vmstat       | Yes                | Statistics from /proc/vmstat. |
| wifi         | Yes                | WiFi device and station statistics. |
| zfs          | Yes                | [ZFS]([http://open-zfs.org/](http://open-zfs.org/)) performance statistics. |
| bonding      | No                 | The number of configured and active slaves of Linux bonding interfaces. |
| buddyinfo    | No                 | Statistics of memory fragments as reported by /proc/buddyinfo. |
| drbd         | No                 | Distributed Replicated Block Device statistics |
| interrupts   | No                 | Detailed interrupts statistics. , OpenBSD |
| ipvs         | No                 | IPVS status from /proc/net/ip_vs and stats from /proc/net/ip_vs_stats. |
| ksmd         | No                 | Kernel and system statistics from /sys/kernel/mm/ksm. |
| logind       | No                 | Session counts from [logind]([http://www.freedesktop.org/wiki/Software/systemd/logind/](http://www.freedesktop.org/wiki/Software/systemd/logind/)). |
| meminfo_numa | No                 | Memory statistics from /proc/meminfo_numa. |
| mountstats   | No                 | Filesystem statistics from /proc/self/mountstats. Exposes detailed NFS client statistics. |
| nfs          | No                 | NFS client statistics from /proc/net/rpc/nfs. This is the same information as nfsstat -c. |
| runit        | No                 | Service status from [runit]([http://smarden.org/runit/](http://smarden.org/runit/)). |
| supervisord  | No                 | Service status from [supervisord]([http://supervisord.org/](http://supervisord.org/)). |
| systemd      | No                 | Service and system status from [systemd]([http://www.freedesktop.org/wiki/Software/systemd/](http://www.freedesktop.org/wiki/Software/systemd/)). |
| tcpstat      | No                 | TCP connection status information from /proc/net/tcp and /proc/net/tcp6. (Warning: the current version has potential performance issues in high load situations.) |
| gmond        | Deprecated         | Statistics from Ganglia |
| megacli      | Deprecated         | RAID statistics from MegaCLI |
| ntp          | Deprecated         | Time drift from an NTP server |


!!! important
    Version added: 1.13.0

    PMM shows NUMA related metrics on the Advanced Data Exploration and NUMA Overview dashboards. To enable this feature, the meminfo_numa option is enabled automatically when you install PMM.
