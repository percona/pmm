
Percona Monitoring and Management Documentation
********************************************************************************

Percona Monitoring and Management (PMM) is an open-source platform
for managing and monitoring MySQL and MongoDB performance.
It is developed by Percona in collaboration with experts
in the field of managed database services, support and consulting.

PMM is a free and open-source solution
that you can run in your own environment
for maximum security and reliability.
It provides thorough time-based analysis for MySQL and MongoDB servers
to ensure that your data works as efficiently as possible.

PMM Concepts
================================================================================

.. toctree::
   :maxdepth: 1

   Architecture Overview <concepts/architecture>
   Security Features <concepts/security>
   

Installation and Configuration
================================================================================

Installing PMM Server
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1

   docker <install/docker>
   Virtual Appliance <install/virtual-appliance>
   Amazon AWS Marketplace <install/ami>

Configuring PMM Server
-------------------------------------------------------------------------------

PMM GUI
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

.. toctree::
   :maxdepth: 1

   Tools of PMM <tool>
   PMM Query Analytics <qan>
   Metrics Monitor <metrics-monitor>

	      
Metrics Monitor Dashboards
================================================================================

.. toctree::
   :maxdepth: 1

    Metrics Monitor Dashboards <index.metrics-monitor.dashboard>

Installing PMM Client
================================================================================

.. toctree::
   :maxdepth: 1

   Installing Clients <install/clients>
   Connecting Clients to the Server  <install/clients.connecting>

Configuring and Administrating PMM Client
================================================================================

.. toctree::
   :maxdepth: 1

   
   Configuring PMM Client <manage/client.config> 
   Getting information about PMM Client <manage/client.info>
   Adding monitoring services <manage/client.add>
   Listing monitoring services <manage/client.list>  
   Removing monitoring services <manage/client.remove>
   Removing orphaned services <manage/client.repair>
   Restarting monitoring services <manage/client.restart>
   Getting passwords used by PMM Client (to security features and/or administration) <manage/client.passwords>
   Starting monitoring services <manage/client.start> 
   Stopping monitoring services <manage/client.stop>
   Cleaning Up Before Uninstall <manage/client.uninstall>
   Purging metrics data <manage/client.purge>
   
Installing and Configuring Services
================================================================================

.. toctree::
   :maxdepth: 1

   Monitoring Service Aliases <manage/client.aliases>

MySQL
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1
	      
   Adding MySQL query analytics service <manage/client.mysql.queries>
   Adding MySQL metrics service <manage/client.mysql.metrics>
   Adding a MySQL or PostgreSQL Remote DB instance to PMM <manage/remote-instance>
   Configuring MySQL for Best Results <conf-mysql>


MongoDB
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1

   Adding MongoDB query analytics service <manage/client.mongodb.queries>
   Adding MongoDB metrics service <manage/client.mongodb.metrics>
   Configuring MongoDB for Monitoring in PMM Query Analytics <conf-mongodb>


ProxySQL
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1

   Adding ProxySQL metrics service <manage/client.proxysql.metrics>

Linux
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1

   Adding general system metrics service <manage/client.linux.metrics>

PostgreSQL
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1
      
   Configuring PostgreSQL for Monitoring <conf-postgres>

Amazon Web Services AWS
--------------------------------------------------------------------------------

.. toctree::
   :maxdepth: 1

   Adding an Amazon RDS DB instance to PMM <manage/amazon-rds>
   
Reference
================================================================================

.. toctree::
   :maxdepth: 1

   Release Notes <release-notes/index>
   Contact Us <contact>
   FAQ <faq>
   Glossaries <index.glossary>


