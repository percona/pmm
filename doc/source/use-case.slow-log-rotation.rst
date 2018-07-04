:orphan: true

.. _use-case.slow-log-rotation:

Use |logrotate| instead of the slow log rotation feature to manage |slow-log|
********************************************************************************

By default, |pmm| manages the slow log for the added |mysql| monitoring service
on the computer where |pmm-client| is installed. This example demonstrates how
to substitute |logrotate| for this default behavior.

.. contents::
   :local:
   :depth: 1

Disable the default behavior of the slow log rotation
================================================================================

The first step is to disable the default slow log rotation when adding the
|mysql| monitoring service.

For this, set the |opt.slow-log-rotation| to *false*.

.. include:: .res/code/sh.org
   :start-after: +pmm-admin.add.mysql.slow-log-rotation+
   :end-before: #+end-block

Check the value of the |gui.slow-logs-rotation| field. It should be *OFF*.

.. seealso::

   More information about monitoring |mysql| in |qan.name|
      :ref:`pmm-admin.add-mysql-queries`

   |qan| settings page
      :ref:`pmm.qan.configuring.settings-tab`

.. important::

   Disabling the slow log rotation feature for an already added |mysql|
   monitoring service is not supported.

   If you already have the |mysql| monitoring service where the slow log
   rotation was not disabled explicitly using the |opt.slow-log-rotation|
   option, remove this monitoring service and add it again setting the
   |opt.slow-log-rotation| to *false*.

Set up |logrotate| to manage the slow log rotation
================================================================================

|logrotate| is a popular utility for managing log files. You can install it
using the package manager (apt or yum, for example) of your |linux|
distribution.

After you add a |mysql| with |opt.slow-log-rotation| set to **false**, you can
run |logrotate| as follows.

|tip.run-this.root|

.. include:: .res/code/sh.org
   :start-after: +logrotate.config_file+
   :end-before: #+end-block

*CONFIG_FILE* is a placeholder for a configuration file that you should supply to
|logrotate| as a mandatory parameter. To use |logrotate| to manage the
|slow-log| for |pmm|, you may supply a file with the following contents.

This is a basic example of |logrotate| for the |mysql| slow logs at 1G for 30
copies (30GB).

.. include:: .res/code/conf.org
   :start-after: +logrotate.slow-log+
   :end-before: #+end-block

.. important::

   In the given example, make sure to set the correct path to the
   |mysql-slow.log| file.

   When running |logrotate| with this example, the effective |mysql| user must
   have the following privileges:
   
   - |opt.reload| privilege in |mysql| so that it can run |sql.flush-slow-logs|
   - |opt.super| privilege so that it can run |sql.set-global|
     |opt.long-query-time|
   
For more information about how to use |logrotate|, refer to its documentation
installed along with the program.

.. admonition:: Related information

   |mysql| Documentation:

      - |sql.flush-slow-logs|: https://dev.mysql.com/doc/refman/8.0/en/flush.html#flush-slow-logs
      - |opt.reload|: https://dev.mysql.com/doc/refman/8.0/en/privileges-provided.html#priv_reload

.. include:: .res/replace.txt
