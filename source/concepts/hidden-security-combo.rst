.. _pmm.security.combining:
		
`Combining Security Features <security.html#pmm-security-combining>`_
================================================================================

You can enable both HTTP password protection and SSL encryption by combining the
corresponding options.

The following example shows how you might :ref:`run the PMM Server container
<server-container>`:

.. include:: ../.res/code/docker.run.example.txt
		 
The following example shows how you might :ref:`connect to PMM Server
<deploy-pmm.client_server.connecting>`:

.. include:: ../.res/code/pmm-admin.config.example.txt

To see which security features are enabled, run either |pmm-admin.ping|,
|pmm-admin.config|, |pmm-admin.info|, or |pmm-admin.list| and look at the server
address field. For example:

.. include:: ../.res/code/pmm-admin.ping.txt
	     
.. include:: ../.res/replace.txt
