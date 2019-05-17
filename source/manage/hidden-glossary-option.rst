.. _pmm.glossary.pmm-server.additional-option:

--------------------------------------------------------------------------------
Using docker environment variables
--------------------------------------------------------------------------------

This glossary contains the addtional parameters that you may pass when starting
|pmm-server|.

Passing options to PMM Server docker at first run
--------------------------------------------------------------------------------

|docker| allows configuration options to be passed using the flag :option:`-e` followed by the option you would like to set.

Here, we pass more than one option to |pmm-server| along with the |docker.run|
command. |tip.run-this.root|.

.. include:: .res/code/docker.run.server-user.example.txt

Passing options to *PMM Server* for an already deployed docker instance
--------------------------------------------------------------------------------

docker doesn't support changing environment variables on an already provisioned container, therefore you need to stop the current container and start a new container with the new options.
variable with **docker start** if you want to change the setting for existing
installation, because **docker start** cares to keep container immutable and
doesn't support changing environment variables. Therefore if you want container
with different properties,  you should run a new container instead.

1. Stop and Rename the old container::

   docker stop pmm-server
   docker rename pmm-server pmm-server-old

2. Ensure you are running the latest version of PMM Server:

      docker pull percona/pmm-server:latest

   .. warning:: When you destroy and recreate the container, all the
      updates you have done through PMM Web interface will be lost. What’s more,
      the software version will be reset to the one in the Docker image. Running
      an old PMM version with a data volume modified by a new PMM version may
      cause unpredictable results. This could include data loss.

4. Start the container with the new settings. For example, changing
   :option:`METRICS_RETENTION` would look as follows::

      docker run -d \
        -p 80:80 \
        --volumes-from pmm-data \
        --name pmm-server \
        --restart always \
        -e METRICS_RESOLUTION=5s \
        percona/pmm-server:latest

5. Once you're satisfied with the new container deployment options and you don't plan to revert, you can remove the old
   container::

     docker rm pmm-server-old

List of |pmm-server| Parameters
--------------------------------------------------------------------------------

.. option:: DISABLE_TELEMETRY

   With :ref:`telemetry <Telemetry>` enabled, your |pmm-server| sends some statistics to
   `v.percona.com`_ every 24 hours. This statistics includes the following
   details:

   - |pmm-server| unique ID
   - |pmm| version
   - The name and version of the operating system
   - |mysql| version
   - |perl| version

   If you do not want your |pmm-server| to send this information, disable telemetry
   when running your |docker| container:

   .. include:: .res/code/docker.run.disable-telemetry.txt

.. option:: METRICS_RETENTION

   This option determines how long metrics are stored at :ref:`PMM
   Server <PMM-Server>`. The value is passed as a combination of hours, minutes, and
   seconds, such as **720h0m0s**. The minutes (a number followed by *m*) and
   seconds (a number followed by *s*) are optional.

   To set the |opt.metrics-retention| option to 8 days, set this option to *192h*.

   |tip.run-this.root|

   .. include:: .res/code/docker.run.e.metrics-retention.txt

   .. seealso::

	 Data retention in PMM
	    :ref:`Data retention <Data-retention>`
	 Queries retention
	    :option:`QUERIES_RETENTION`

.. option:: QUERIES_RETENTION

   This option determines how many days queries are stored at :ref:`PMM Server <PMM-Server>`. 

   .. include:: .res/code/docker.run.e.queries-retention.txt

   .. seealso::

	 Metrics retention
	    :option:`METRICS_RETENTION`
	 Data retention in PMM
	    :ref:`Data retention <Data-retention>`

.. option:: ORCHESTRATOR_ENABLED

   This option enables |orchestrator| (See
   :ref:`pmm.using.orchestrator`). By default it is disabled. It is
   also desabled if this option contains **false**.

   .. include:: .res/code/docker.run.orchestrator-enabled.txt

   .. seealso::

	 Orchestrator
	    :ref:`Orchestrator <Orchestrator>`
	 Orchestrator Credentials
	    - :option:`ORCHESTRATOR_USER`
	    - :option:`ORCHESTRATOR_PASSWORD`

.. option:: ORCHESTRATOR_USER

   Pass this option, when running your :ref:`PMM Server <PMM-Server>` via
   |docker| to set the orchestrator user. You only need this
   parameter (along with :option:`ORCHESTRATOR_PASSWORD` if you have set up a custom
   |orchestrator| user.

   This option has no effect if the
   :option:`ORCHESTRATOR_ENABLED` option is
   set to **false**.

   .. include:: .res/code/docker.run.orchestrator-enabled.orchestrator-user.orchestrator-password.txt

.. option:: ORCHESTRATOR_PASSWORD

   Pass this option, when running your :ref:`PMM Server <PMM-Server>` via |docker| to set
   the orchestrator password.

   This option has no effect if the
   :option:`ORCHESTRATOR_ENABLED`
   option is set to **false**.

   .. include:: .res/code/docker.run.orchestrator-enabled.orchestrator-user.orchestrator-password.txt

   .. seealso:: :option:`ORCHESTRATOR_ENABLED`

.. option:: SERVER_USER

   By default, the user name is ``pmm``. Use this option to use another user
   name.

   |tip.run-this.root|.

   .. include:: .res/code/docker.run.server-user.txt

.. option:: SERVER_PASSWORD

   Set the password to access the |pmm-server| web interface.

   |tip.run-this.root|.

   .. include:: .res/code/docker.run.server-password.txt
   
   By default, the user name is ``pmm``. You can change it by passing the
   :option:`SERVER_USER` variable.

.. option:: METRICS_RESOLUTION

   This environment variable sets the minimum resolution for checking
   metrics. You should set it if the latency is higher than 1 second.

   |tip.run-this.root|.

   .. include:: .res/code/docker.run.metrics-resolution.txt

.. option:: METRICS_MEMORY

   By default, |prometheus| in |pmm-server| uses all available memory for
   storing the most recently used data chunks.  Depending on the amount of
   data coming into |prometheus|, you may require to allow less memory
   consumption if it is needed for other processes.

   .. include:: .res/contents/important.option.metrics-memory.txt

   If you are still using a version of |pmm| prior to 1.13 you might need to
   set the metrics memory by passing the |opt.metrics-memory| environment
   variable along with the |docker.run| command.

   |tip.run-this.root|. The value must be passed in kilobytes. For example,
   to set the limit to 4 GB of memory run the following command:

   .. include:: .res/code/docker.run.metrics-memory.txt

   .. seealso:: 

	 |docker| documentation: Controlling memory usage in a |docker| container
	    https://docs.docker.com/config/containers/resource_constraints/

.. option:: DISABLE_UPDATES

   To update your |pmm| from web interface you only need to click the
   |gui.update| on the home page. The |opt.disable-updates| option is useful
   if updating is not desirable. Set it to **true** when running |pmm| in
   the |docker| container.

   |tip.run-this.root|.

   .. include:: .res/code/docker.run.disable-updates.txt

   The |opt.disable-updates| option removes the |gui.update| button
   from the interface and prevents the system from being updated manually.

.. _v.percona.com: http://v.percona.com

.. include:: .res/replace.txt
