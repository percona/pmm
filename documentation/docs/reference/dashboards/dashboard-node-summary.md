# Node Summary

![!image](../../images/PMM_Node_Summary.jpg)

## Load Average

The average number of processes that are running or waiting for CPU. The value turns orange when the load reaches the number of virtual CPUs, and red at twice that number.

## System Summary

The output from ``pt-summary``, one of the [Percona Toolkit utilities](https://www.percona.com/doc/percona-toolkit/LATEST/pt-summary.html). It shows the report for the node that you select in the **Node Name** list.

## Service Types

The number of monitored services on the node, by type. Arbiters and services that are down are counted too.

## Services

Whether each monitored service on the node is running. The **Status** column shows **UP** when the service is reachable, and **DOWN** when it isn't. For external services and HAProxy, **UP** means that PMM can collect metrics from their exporter. If the node has no monitored services, the panel shows **No services monitored on this node**.

## CPU Usage

The CPU time is measured in clock ticks or seconds. It is useful to measure CPU time as a percentage of the CPU's capacity, which is called the CPU usage.

## CPU Saturation and Max Core Usage

When a system is running with maximum CPU utilization, the transmitting and receiving threads must all share the available CPU. This will cause data to be queued more frequently to cope with the lack of CPU. CPU Saturation may be measured as the length of a wait queue, or the time spent waiting on the queue.

## Interrupts and Context Switches

Interrupt is an input signal to the processor indicating an event that needs immediate attention. An interrupt signal alerts the processor and serves as a request for the processor to interrupt the currently executing code, so that the event can be processed in a timely manner.

Context switch is the process of storing the state of a process or thread, so that it can be restored and resume execution at a later point. This allows multiple processes to share a single CPU, and is an essential feature of a multitasking operating system.


## Swap Activity

Swap Activity is memory management that involves swapping sections of memory to and from physical storage.

## I/O Activity

Disk I/O includes read or write or input/output operations involving a physical disk. It is the speed with which the data transfer takes place between the hard disk drive and RAM.

## Disk Operations

The number of read and write operations per second on the node's disks. Reads are shown above the axis and writes below it. Sudden spikes often mean that the disks are overloaded. LVM and software RAID devices (`dm-*` and `md*`) aren't included, so the same operation isn't counted twice.

## Disk IO Latency

Shows average latency for Reads and Writes IO Devices.  Higher than typical latency for highly loaded storage indicates saturation (overload) and is frequent cause of performance problems.  Higher than normal latency also can indicate internal storage problems.

## Disk IO Utilization

The percentage of time that each disk was busy with at least one I/O request. It's the same value as `%util` in `iostat`. The higher it is, the more likely requests are to wait in a queue, so check it together with **Disk IO Latency** and **Disk IO Load**.

## Disk IO Load

Shows how much disk was loaded for reads or writes as average number of outstanding requests at different period of time.  High disk load is a good measure of actual storage utilization. Different storage types handle load differently - some will show latency increases on low loads others can handle higher load with no problems.

## Global File Descriptors Usage

The number of files open on the node. If the system has no real limit, the limit isn't shown, so the chart stays readable.

## Network Traffic

Network traffic refers to the amount of data moving across a network at a given point in time.


## Local Network Errors

Total Number of Local Network Interface Transmit Errors, Receive Errors and Drops.  Should be  Zero

## TCP Retransmission

Retransmission, essentially identical with Automatic repeat request (ARQ), is the resending of packets which have been either damaged or lost. Retransmission is one of the basic mechanisms used by protocols operating over a packet switched computer network to provide reliable communication (such as that provided by a reliable byte stream, for example TCP).
