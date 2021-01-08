# Disk space

## Mountpoint Usage

This metric shows the percentage of disk space utilization for every mountpoint defined on the system. It is not good having some of the mountpoints close to 100% of space utilization, the risk is to have a *disk full* error that can block one of the services or even causing a crash of the entire sytem.

In case a mountpoint is close to 100%, consider to cancel unused files or to expand the space allocate to it.

## Mountpoint

This metric shows information about the disk space usage of the specified mountpoint.

Used
: Is the amount of space used

Free
: Is the amount if space not in use

The total disk space allocated to the mountpoint is the sum of *Used* and *Free* space.

It is not good having *Free* close to 0 B. The risk is to have a *disk full* error that can block one of the services or even causing a crash of the entire system.

In case *Free* is close to 0 B, consider to cancel unused files or to expand the space allocated to the mountpoint.
