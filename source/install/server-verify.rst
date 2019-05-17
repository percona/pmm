.. _deploy-pmm.server-verifying:

--------------------------------------------------------------------------------
`Verifying PMM Server <index.html#deploy-pmm-server-verifying>`_
--------------------------------------------------------------------------------

In your browser, go to the server by its IP address. If you run your server as a
virtual appliance or by using an |amazon| machine image, you will need to setup
the user name, password and your public key if you intend to connect to the
server by using ssh. This step is not needed if you run |pmm-server| using
|docker|.

In the given example, you would need to direct your browser to
*http://192.168.100.1*. Since you have not added any monitoring services yet,
the site will not show any data.

.. _deploy-pmm.table.web-interface.component.access:

.. table:: Accessing the Components of the Web Interface

   ==================================== ======================================
   Component                            URL
   ==================================== ======================================
   :term:`PMM Home Page`                ``http://192.168.100.1``
   :term:`Metrics Monitor (MM)`         | ``http://192.168.100.1/graph/``
                                        | User name: ``admin``
                                        | Password: ``admin``
   Orchestrator                         ``http://192.168.100.1/orchestrator``
   ==================================== ======================================

You can also check if |pmm-server| is available requesting the /ping
URL as in the following example:

.. include:: ../.res/code/curl.ping.txt

.. include:: ../.res/replace.txt
