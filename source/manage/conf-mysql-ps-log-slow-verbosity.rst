`log_slow_verbosity <pmm.conf-mysql.log-slow-verbosity>`_
--------------------------------------------------------------------------------

|log_slow_verbosity|_ variable specifies how much information to include in
the slow query log. It is a good idea to set |log_slow_verbosity|_ to ``full``
so that maximum amount of information about each captured query is stored.

.. seealso::

   |mysql| Documentation
      `Setting variables <https://dev.mysql.com/doc/refman/5.7/en/set-variable.html>`_

.. |long_query_time| replace:: ``long_query_time``
.. _long_query_time: http://dev.mysql.com/doc/refman/5.7/en/server-system-variables.html#sysvar_long_query_time

.. |log_slow_rate_limit| replace:: ``log_slow_rate_limit``
.. _log_slow_rate_limit: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_limit

.. |log_slow_rate_type| replace:: ``log_slow_rate_type``
.. _log_slow_rate_type: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_rate_type

.. |log_slow_verbosity| replace:: ``log_slow_verbosity``
.. _log_slow_verbosity: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#log_slow_verbosity

.. |slow_query_log_always_write_time| replace:: ``slow_query_log_always_write_time``
.. _slow_query_log_always_write_time: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_always_write_time

.. |slow_query_log_use_global_control| replace:: ``slow_query_log_use_global_control``
.. _slow_query_log_use_global_control: https://www.percona.com/doc/percona-server/5.7/diagnostics/slow_extended.html#slow_query_log_use_global_control

.. include:: ../.res/replace.txt
