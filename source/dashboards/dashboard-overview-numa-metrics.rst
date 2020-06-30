.. _dashboard-overview-numa-metrics:

############
NUMA Details
############

For each node, this dashboard shows metrics related to Non-uniform memory
access (NUMA).


.. _dashboard-overview-numa-metrics.memory-usage:
.. _memory-usage:

************
Memory Usage
************

Remotes over time the total, used, and free memory.

.. _dashboard-overview-numa-metrics.free-memory-percent:
.. _free-memory-percent:

*******************
Free Memory Percent
*******************

Shows the free memory as the ratio to the total available memory.


.. _dashboard-overview-numa-metrics.numa-memory-usage-types:
.. _numa-memory-usage-types:

***********************
NUMA Memory Usage Types
***********************

Dirty
   Memory waiting to be written back to disk
Bounce
   Memory used for block device bounce buffers
Mapped
   Files which have been mmaped, such as libraries

KernelStack The memory the kernel stack uses. This is not reclaimable.


.. _dashboard-overview-numa-metrics.numa-allocation-hits:
.. _numa-allocation-hits:

********************
NUMA Allocation Hits
********************

Memory successfully allocated on this node as intended.


.. _dashboard-overview-numa-metrics.numa-allocation-missed:
.. _numa-allocation-missed:

**********************
NUMA Allocation Missed
**********************

Memory missed is allocated on a node despite the process preferring some different node.

Memory foreign is intended for a node, but actually allocated on some different node.


.. _dashboard-overview-numa-metrics.anonymous-memory:
.. _anonymous-memory:

****************
Anonymous Memory
****************

Active
   Anonymous memory that has been used more recently and usually not swapped out.
Inactive
   Anonymous memory that has not been used recently and can be swapped out.



.. _dashboard-overview-numa-metrics.numa-file-page-cache:
.. _numa-file-page-cache:

*********************
NUMA File (PageCache)
*********************

Active(file) Pagecache memory that has been used more recently and usually not
reclaimed until needed.

Inactive(file) Pagecache memory that can be reclaimed without huge performance
impact.



.. _dashboard-overview-numa-metrics.shared-memory:
.. _shared-memory:

*************
Shared Memory
*************

Shmem Total used shared memory (shared between several processes, thus including
RAM disks, SYS-V-IPC and BSD like SHMEM).


.. _dashboard-overview-numa-metrics.hugepages-statistics:
.. _hugepages-statistics:

********************
HugePages Statistics
********************


Total
   Number of hugepages being allocated by the kernel (Defined with ``vm.nr_hugepages``).
Free
   The number of hugepages not being allocated by a process
Surp
  The number of hugepages in the pool above the value in ``vm.nr_hugepages``. The
  maximum number of surplus hugepages is controlled by
  ``vm.nr_overcommit_hugepages``.


.. _dashboard-overview-numa-metrics.local-processes:
.. _local-processes:

***************
Local Processes
***************

Memory allocated on a node while a process was running on it.


.. _dashboard-overview-numa-metrics.remote-processes:
.. _remote-processes:

****************
Remote Processes
****************

Memory allocated on a node while a process was running on some other node.


.. _dashboard-overview-numa-metrics.slab-memory:
.. _slab-memory:

***********
Slab Memory
***********

Slab
   Allocation is a memory management mechanism intended for the efficient memory allocation of kernel objects.
SReclaimable
   The part of the Slab that might be reclaimed (such as caches).
SUnreclaim
   The part of the Slab that can't be reclaimed under memory pressure

