.. _pmm-admin.check-network:

--------------------------------------------------------------------------------
`Checking the network <pmm-admin.html#pmm-admin-check-network>`_
--------------------------------------------------------------------------------

Use the |pmm-admin.check-network| command to run tests
that verify connectivity between |pmm-client| and |pmm-server|.

.. _pmm-admin.check-network.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.check-network.options:

.. include:: ../.res/code/pmm-admin.check-network.options.txt
		
.. _pmm-admin.check-network.options:

.. rubric:: OPTIONS

The |pmm-admin.check-network| command does not have its own options,
but you can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.check-network.detailed-description:

.. rubric:: DETAILED DESCRIPTION

Connection tests are performed both ways, with results separated accordingly:

* ``Client --> Server``

  Pings |consul| API, |qan.name| API, and |prometheus| API
  to make sure they are alive and reachable.

  Performs a connection performance test to see the latency
  from |pmm-client| to |pmm-server|.

* ``Client <-- Server``

  Checks the status of |prometheus| endpoints
  and makes sure it can scrape metrics from corresponding exporters.

  Successful pings of |pmm-server| from |pmm-client|
  do not mean that Prometheus is able to scrape from exporters.
  If the output shows some endpoints in problem state,
  make sure that the corresponding service is running
  (see |pmm-admin.list|_).
  If the services that correspond to problematic endpoints are running,
  make sure that firewall settings on the |pmm-client| host
  allow incoming connections for corresponding ports.

.. _pmm-admin.check-network.output-example:

.. rubric:: OUTPUT EXAMPLE

.. _code.pmm-admin.check-network.output:

.. include:: ../.res/code/pmm-admin.check-network.output.txt

For more information, run
|pmm-admin.check-network|
|opt.help|.

.. include:: ../.res/replace.txt
