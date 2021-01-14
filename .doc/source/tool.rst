.. _using:

Tools of |pmm|
********************************************************************************

You can access the |pmm| web interface using the IP address of the host where
|pmm-server| is running.  For example, if |pmm-server| is running on a host with
IP 192.168.100.1, access the following address with your web browser:
``http://192.168.100.1``.

.. seealso::

   Installing |pmm-server|
      :ref:`deploy-pmm.server.installing`

The |pmm| home page that opens provides an overview of the environment that you
have set up to monitor by using the |pmm-admin| tool.

From the |pmm| home page, you can access specific monitoring tools, or
dashboards. Each dashboard features a collection of metrics. These are graphs of
a certain type that represent one specific aspect showing how metric values
change over time.

.. figure:: .res/graphics/png/pmm.home-page.png

   The home page is an overview of your system

By default the |pmm| home page lists most recently used dashboards and helpful
links to the information that may be useful to understand |pmm| better.

The |pmm| home page lists all hosts that you have set up for monitoring as well
as the essential details about their performance such as CPU load, disk
performance, or network activity.

.. rubric:: More about |pmm| Components

.. toctree::
   :maxdepth: 2
   
   qan
   metrics-monitor

.. include:: .res/replace.txt
