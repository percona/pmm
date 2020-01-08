.. _install-client-apt:

Installing DEB packages using apt-get
================================================================================

If you are running a DEB-based |linux| distribution, use the |apt| package
manager to install |pmm-client| from the official Percona software repository.

|percona| provides :file:`.deb` packages for 64-bit versions of the following
distributions:

.. include:: ../.res/contents/list.pmm-client.supported-apt-platform.txt

.. note::

   |pmm-client| should work on other DEB-based distributions, but it is tested
   only on the platforms listed above.

To install the |pmm-client| package, complete the following
procedure. |tip.run-all.root|:

1. Configure |percona| repositories using the `percona-release <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ tool. First you’ll need to download and install the official percona-release package from Percona::

     wget https://repo.percona.com/apt/percona-release_latest.generic_all.deb
     sudo dpkg -i percona-release_latest.generic_all.deb

   .. raw:: html

      <script id="asciicast-LaIiFlGWZdWAMPf4p4OUEHrjB" src="https://asciinema.org/a/LaIiFlGWZdWAMPf4p4OUEHrjB.js" async data-theme="solarized-light" data-rows="8"></script>

   .. note:: If you have previously enabled the experimental or testing
      Percona repository, don't forget to disable them and enable the release
      component of the original repository as follows::

         sudo percona-release disable all
         sudo percona-release enable original release

   See `percona-release official documentation <https://www.percona.com/doc/percona-repo-config/percona-release.html>`_ for details.

#. Install the ``pmm2-client`` package::

     sudo apt-get update
     sudo apt-get install pmm2-client

   .. raw:: html

      <script id="asciicast-ZBfCORUanwrZMPD3hkiHYKBkv" src="https://asciinema.org/a/ZBfCORUanwrZMPD3hkiHYKBkv.js" async data-theme="solarized-light" data-rows="8"></script>

#. Once PMM Client is installed, run the ``pmm-admin config`` command with your PMM Server IP address to register your Node within the Server:

   .. include:: ../.res/code/pmm-admin.config.server.url.dummy.txt

   You should see the following::

     Checking local pmm-agent status...
     pmm-agent is running.
     Registering pmm-agent on PMM Server...
     Registered.
     Configuration file /usr/local/percona/pmm-agent.yaml updated.
     Reloading pmm-agent configuration...
     Configuration reloaded.

.. include:: ../.res/replace.txt
