.. _deploy-pmm.server-verifying:

--------------------------------------------------------------------------------
`Verifying PMM Server <index.html#deploy-pmm-server-verifying>`_
--------------------------------------------------------------------------------

In your browser, go to the server by its IP address. If you run your server as a
virtual appliance or by using an Amazon machine image, you will need to setup
the user name, password and your public key if you intend to connect to the
server by using ssh. This step is not needed if you run PMM Server using
Docker.

In the given example, you would need to direct your browser to
*http://192.168.100.1*. Since you have not added any monitoring services yet,
the site will show only data related to the PMM Server internal services.

.. _deploy-pmm.table.web-interface.component.access:

.. rubric:: Accessing the Components of the Web Interface

- ``http://192.168.100.1`` to access :ref:`dashboard-home`.

- ``http://192.168.100.1/graph/`` to access :ref:`Metrics Monitor <pmm-metrics-monitor>`.

- ``http://192.168.100.1/swagger/`` to access :ref:`PMM API <pmm-server-api>`.

PMM Server provides user access control, and therefore you will need
user credentials to access it:

.. image:: /_images/pmm-login-screen.png

The default user name is ``admin``, and the default password is ``admin`` also.
You will be proposed to change the default password at login if you didn't it.

.. note:: You will use the same credentials at `connecting <https://www.percona.com/doc/percona-monitoring-and-management/2.x/manage/client-config.html>`_ your PMM Client to PMM Server.
