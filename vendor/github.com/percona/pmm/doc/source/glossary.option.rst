.. _pmm/glossary.option:

================================================================================
|pmm-server| Options
================================================================================

This glossary contains the options that you may pass as additional parameters
when starting |pmm-server|.

If you use |docker| to run the server, use the :option:`-e` flag followed by the
parameter. Use this flag in front of each parameter that you pass to the
|pmm-server|.

Here, we pass more than one option to |pmm-server| along with the |docker.run|
command. |tip.run-this.root|.

.. include:: .res/code/sh.org
   :start-after: +docker.run.server-user.example+
   :end-before: #+end-block

List of |pmm-server| Options
================================================================================

.. glossary::
   :sorted:

   METRICS_RETENTION (Option)

      This option determines how long metrics are stored at :term:`PMM
      Server`. The value is passed as a combination of hours, minutes, and
      seconds, such as **720h0m0s**. The minutes (a number followed by *m*) and
      seconds (a number followed by *s*) are optional.

      To set the |opt.metrics-retention| option to 8 days, set this option to *192h*.

      |tip.run-this.root|

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.e.metrics-retention+
	 :end-before: #+end-block

      .. seealso::

	 Data retention in PMM

	    :term:`Data retention`

	 Queries retention

	    :term:`QUERIES_RETENTION <QUERIES_RETENTION (Option)>`

   QUERIES_RETENTION (Option)

      This option determines how many days queries are stored at :term:`PMM Server`. 

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.e.queries-retention+
	 :end-before: #+end-block

      .. seealso::

	 Metrics retention

	    :term:`METRICS_RETENTION <METRICS_RETENTION (Option)>`

	 Data retention in PMM

	    :term:`Data retention`

   ORCHESTRATOR_ENABLED (Option)

      This option enables |orchestrator| (See
      :ref:`pmm/using.orchestrator`). By default it is disabled. It is
      also desabled if this option contains **false**.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.orchestrator-enabled+
	 :end-before: #+end-block

      .. seealso::

	 Orchestrator
	    :term:`Orchestrator`
	 Orchestrator Credentials
	    - :term:`ORCHESTRATOR_USER <ORCHESTRATOR_USER (Option)>`
	    - :term:`ORCHESTRATOR_PASSWORD <ORCHESTRATOR_PASSWORD (Option)>`

   ORCHESTRATOR_USER (Option)

      Pass this option, when running your :term:`PMM Server` via
      |docker| to set the orchestrator user. You only need this
      parameter (along with :term:`ORCHESTRATOR_PASSWORD
      <ORCHESTRATOR_PASSWORD (Option)>` if you have set up a custom
      |orchestrator| user.

      This option has no effect if the
      :term:`ORCHESTRATOR_ENABLED <ORCHESTRATOR_ENABLED (Option)>` option is
      set to **false**.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.orchestrator-enabled.orchestrator-user.orchestrator-password+
	 :end-before: #+end-block

   ORCHESTRATOR_PASSWORD (Option)

      Pass this option, when running your :term:`PMM Server` via |docker| to set
      the orchestrator password.

      This option has no effect if the
      :term:`ORCHESTRATOR_ENABLED <ORCHESTRATOR_ENABLED (Option)>`
      option is set to **false**.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.orchestrator-enabled.orchestrator-user.orchestrator-password+
	 :end-before: #+end-block

      .. seealso:: :term:`ORCHESTRATOR_ENABLED <ORCHESTRATOR_ENABLED (Option)>`

   SERVER_USER (Option)

      By default, the user name is ``pmm``. Use this option to use another user
      name.

      |tip.run-this.root|.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.server-user+
	 :end-before: #+end-block

   SERVER_PASSWORD (Option)

      Set the password to access the |pmm-server| web interface.

      |tip.run-this.root|.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.server-password+
	 :end-before: #+end-block
      
      By default, the user name is ``pmm``. You can change it by passing the
      :term:`SERVER_USER <SERVER_USER (Option)>` variable.

   METRICS_RESOLUTION (Option)

      This option sets the minimum resolution for checking metrics. You should
      set it if the latency is higher than 1 second.

      |tip.run-this.root|.

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.metrics-resolution+
	 :end-before: #+end-block


   METRICS_MEMORY (Option)

      By default, |prometheus| in |pmm-server| uses up to 768 MB of memory for storing
      the most recently used data chunks.  Depending on the amount of data coming into
      |prometheus|, you may require a higher limit to avoid throttling data ingestion,
      or allow less memory consumption if it is needed for other processes.

      |tip.run-this.root|. The value must be passed in kilobytes. For example,
      to set the limit to 4 GB of memory run the following command:

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.metrics-memory+
	 :end-before: #+end-block

      .. note::

	 The limit affects only memory reserved for data chunks.  Actual RAM
	 usage by Prometheus is higher.  It is recommended to set this limit to
	 roughly 2/3 of the total memory that you are planning to allow for
	 Prometheus. For example, if you set the limit to 4 GB, then
	 |prometheus| will use up to 6 GB of memory.

   DISABLE_UPDATES (Option)

      The |opt.disable-updates| option removes the :guilabel:`Update` button
      from the interface and prevents the system from being updated manually.

      Set it to *true* when running a docker container:

      .. include:: .res/code/sh.org
	 :start-after: +docker.run.disable-updates+
	 :end-before: #+end-block

.. include:: .res/replace/name.txt
.. include:: .res/replace/option.txt
.. include:: .res/replace/program.txt
.. include:: .res/replace/fragment.txt
