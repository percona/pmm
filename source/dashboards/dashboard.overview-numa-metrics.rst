.. _dashboard.overview-numa-metrics:

|dbd.overview-numa-metrics| Dashboard
================================================================================

For each node, this dashboard shows metrics related to Non-uniform memory
access (NUMA).

.. contents::
   :local:

..note: 

    Users who already have `General system metrics service <https://www.percona.com/doc/percona-monitoring-and-management/pmm-admin.html#pmm-admin-add-linux-metrics>`_ monitored and would like to add NUMA metrics need to remove and re-add ``linux:metrics`` on the node::
           
       pmm-admin remove linux:metrics
       pmm-admin add linux:metrics

.. _dashboard.overview-numa-metrics.memory-usage:

:ref:`Memory Usage <dashboard.overview-numa-metrics.memory-usage>`
--------------------------------------------------------------------------------

Remotes over time the total, used, and free memory.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.free-memory-percent:

:ref:`Free Memory Percent <dashboard.overview-numa-metrics.free-memory-percent>`
--------------------------------------------------------------------------------

Shows the free memory as the ratio to the total available memory.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.numa-memory-usage-types:

:ref:`NUMA Memory Usage Types <dashboard.overview-numa-metrics.numa-memory-usage-types>`
----------------------------------------------------------------------------------------

Dirty
   Memory waiting to be written back to disk
Bounce
   Memory used for block device bounce buffers
Mapped
   Files which have been mmaped, such as libraries

KernelStack The memory the kernel stack uses. This is not reclaimable.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.numa-allocation-hits:

:ref:`NUMA Allocation Hits <dashboard.overview-numa-metrics.numa-allocation-hits>`
----------------------------------------------------------------------------------

Memory successfully allocated on this node as intended.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.numa-allocation-missed:

:ref:`NUMA Allocation Missed <dashboard.overview-numa-metrics.numa-allocation-missed>`
--------------------------------------------------------------------------------------

Memory missed is allocated on a node despite the process preferring some different node.

Memory foreign is intended for a node, but actually allocated on some different node.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.anonymous-memory:

:ref:`Anonymous Memory <dashboard.overview-numa-metrics.anonymous-memory>`
--------------------------------------------------------------------------------

Active
   Anonymous memory that has been used more recently and usually not swapped out.
Inactive
   Anonymous memory that has not been used recently and can be swapped out.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.numa-file-page-cache:

:ref:`NUMA File (PageCache) <dashboard.overview-numa-metrics.numa-file-page-cache>`
-----------------------------------------------------------------------------------

Active(file) Pagecache memory that has been used more recently and usually not
reclaimed until needed.

Inactive(file) Pagecache memory that can be reclaimed without huge performance
impact.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.shared-memory:

:ref:`Shared Memory <dashboard.overview-numa-metrics.shared-memory>`
--------------------------------------------------------------------------------

Shmem Total used shared memory (shared between several processes, thus including
RAM disks, SYS-V-IPC and BSD like SHMEM)

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.hugepages-statistics:

:ref:`HugePages Statistics <dashboard.overview-numa-metrics.hugepages-statistics>`
----------------------------------------------------------------------------------

Total
   Number of hugepages being allocated by the kernel (Defined with vm.nr_hugepages).
Free
   The number of hugepages not being allocated by a process
Surp
  The number of hugepages in the pool above the value in vm.nr_hugepages. The
  maximum number of surplus hugepages is controlled by
  vm.nr_overcommit_hugepages.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.local-processes:

:ref:`Local Processes <dashboard.overview-numa-metrics.local-processes>`
--------------------------------------------------------------------------------

Memory allocated on a node while a process was running on it.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.remote-processes:

:ref:`Remote Processes <dashboard.overview-numa-metrics.remote-processes>`
--------------------------------------------------------------------------------

Memory allocated on a node while a process was running on some other node.

|view-all-metrics| |this-dashboard|

.. _dashboard.overview-numa-metrics.slab-memory:

:ref:`Slab Memory <dashboard.overview-numa-metrics.slab-memory>`
--------------------------------------------------------------------------------

Slab
   Allocation is a memory management mechanism intended for the efficient memory allocation of kernel objects.
SReclaimable
   The part of the Slab that might be reclaimed (such as caches).
SUnreclaim
   The part of the Slab that can't be reclaimed under memory pressure

|view-all-metrics| |this-dashboard|

.. |this-dashboard| replace:: :ref:`dashboard.overview-numa-metrics`

.. include:: ../.res/replace.txt
