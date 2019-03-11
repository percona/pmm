.. _pmm.security.password-protection.enabling:

:ref:`Enabling Password Protection <pmm.security.password-protection.enabling>`
================================================================================

You can set the password for accessing the |pmm-server| web interface by passing
the :option:`SERVER_PASSWORD` environment variable when
:ref:`creating and running the PMM Server container <server-container>`.

To set the environment variable, use the ``-e`` option.

By default, the user name is ``pmm``. You can change it by passing the
following example uses an insecure port 80 which is typically used for HTTP
connections.

|tip.run-all.root|.

.. include:: ../.res/code/docker.run.server-user.example.txt

|pmm-client| uses the same credentials to communicate with |pmm-server|.  If you
set the user name and password as described, specify them when :ref:`connecting
a PMM Client to a PMM Server <deploy-pmm.client_server.connecting>`:

.. include:: ../.res/code/pmm-admin.config.server.server-user.server-password.txt

.. include:: ../.res/replace.txt
