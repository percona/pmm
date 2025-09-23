# MongoDB Swap Allocation

## Description
This check warns if there is no swap memory allocated to your instance.

## Resolution

MongoDB performs best if swapping can be avoided or kept to a minimum since retrieving data from swap on disk will always be slower than accessing data in RAM. However, if the system hosting MongoDB runs out of RAM, swapping can prevent the Linux OOM (Out of memory) Killer from terminating the mongod process.

Choose one of the following swap strategies:


- Assign swap space on your system, and configure the kernel to only permit swapping under high-memory load, or


- Do not assign swap space on your system, and configure the kernel to disable swapping entirely.


If your MongoDB instance is hosted on a system that runs other software, you should choose the first swap strategy to prevent possible negative impacts on those applications. Do not disable swap in this case. Percona highly recommends that you run MongoDB on its own dedicated system whenever possible.

Once the swap space is allocated as per the recommendation, configure it by setting the **vm.swappiness** parameter.

**Set vm.swappiness:**

“Swappiness” is a Linux kernel setting that influences the behavior of the Virtual Memory manager. MongoDB performs best where swapping can be avoided or kept to a minimum. As such you should set vm.swappiness to 1, which permits the kernel to swap only to avoid out-of-memory problems.

- Edit the **/etc/sysctl.conf** file and add the following line:

> vm.swappiness = 1

- Run the following command to apply the setting:

> sudo sysctl -p


## Need more support from Percona?
Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
