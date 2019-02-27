.. _pmm-admin.show-passwords:

:ref:`Getting passwords used by PMM Client <pmm-admin.show-passwords>`
================================================================================

Use the |pmm-admin.show-passwords| command to print credentials stored in the
configuration file (by default: :file:`/usr/local/percona/pmm-client/pmm.yml`).

.. _pmm-admin.show-passwords.usage:

.. rubric:: USAGE

|tip.run-this.root|

.. _code.pmm-admin.show-passwords.options:

.. include:: ../.res/code/pmm-admin.show-passwords.options.txt

.. _pmm-admin.show-passwords.options:

.. rubric:: OPTIONS

The |pmm-admin.show-passwords| command does not have its own options, but you
can use :ref:`global options that apply to any other command
<pmm-admin.options>`

.. _pmm-admin.show-passwords.output:

.. rubric:: OUTPUT

This command prints HTTP authentication credentials and the password for the
``pmm`` user that is created on the |mysql| instance if you specify the
|opt.create-user| option when :ref:`adding a service <pmm-admin.add>`.

|tip.run-this.root|

.. _code.pmm-admin.show-passwords:

.. include:: ../.res/code/pmm-admin.show-passwords.txt

For more information, run |pmm-admin.show-passwords|  |opt.help|.

.. include:: ../.res/replace.txt
