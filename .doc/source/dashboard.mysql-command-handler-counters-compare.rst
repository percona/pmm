.. _dashboard.mysql-command-handler-counters-compare:

|dbd.mysql-command-handler-counters-compare| Dashboard
================================================================================

This dashboard shows server status variables. On this dashboard, you may select
multiple servers and compare their counters simultaneously.

Server status variables appear in two sections: *Commands* and
*Handlers*. Choose one or more variables in the *Command* and *Handler* fields
in the top menu to select the variables which will appear in the *COMMANDS* or
*HANDLERS* section for each host. Your comparison may include from one up to
three hosts.

By default or if no item is selected in the menu, |pmm| displays each command or
handler respectively.

.. seealso::

   |mysql| Documentation: Server Status Variables
      https://dev.mysql.com/doc/refman/8.0/en/server-status-variables.html

.. include:: .res/replace.txt
