`Executing Custom Queries <pmm.conf-mysql.executing.custom.queries>`_
================================================================================

Starting from the version 1.15.0, |pmm| provides user the ability to take a SQL
``SELECT`` statement and turn the result set into metric series in |pmm|. The
queries are executed at the LOW RESOLUTION level, which by default is every 60
seconds. A key advantage is that you can extend |pmm| to profile metrics unique
to your environment (see users table example below), or to introduce support
for a table that isn't part of |pmm| yet. This feature is on by default and only
requires that you edit the configuration file and use vaild YAML syntax. The
default configuration file location is
``/usr/local/percona/pmm-client/queries-mysqld.yml``.

.. rubric:: Example - Application users table

We're going to take a users table of upvotes and downvotes and turn this into
two metric series, with a set of labels. Labels can also store a value. You can
filter against labels.

.. rubric:: Browsing metrics series using Advanced Data Exploration Dashboard

Lets look at the output so we understand the goal - take data from a |mysql|
table and store in |pmm|, then display as a metric series. Using the Advanced
Data Exploration Dashboard you can review your metric series. 

.. rubric:: MySQL table

Lets assume you have the following users table that includes true/false, string,
and integer types.

.. code-block:: bash

   SELECT * FROM `users`
   +----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+
   | id | app  | user_type    | last_name | first_name | logged_in | active_subscription | banned | upvotes | downvotes |
   +----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+
   |  1 | app2 | unprivileged | Marley    | Bob        |         1 |                   1 |      0 |     100 |        25 |
   |  2 | app3 | moderator    | Young     | Neil       |         1 |                   1 |      1 |     150 |        10 |
   |  3 | app4 | unprivileged | OConnor   | Sinead     |         1 |                   1 |      0 |      25 |        50 |
   |  4 | app1 | unprivileged | Yorke     | Thom       |         0 |                   1 |      0 |     100 |       100 |
   |  5 | app5 | admin        | Buckley   | Jeff       |         1 |                   1 |      0 |     175 |         0 |
   +----+------+--------------+-----------+------------+-----------+---------------------+--------+---------+-----------+

.. rubric:: Explaining the YAML syntax

We'll go through a simple example and mention what's required for each line. The
metric series is constructed based on the first line and appends the column name
to form metric series. Therefore the number of metric series per table will be
the count of columns that are of type ``GAUGE`` or ``COUNTER``. This metric
series will be called ``app1_users_metrics_downvotes``:

.. code-block:: bash

   app1_users_metrics:                                 ## leading section of your metric series.
     query: "SELECT * FROM app1.users"                 ## Your query. Don't forget the schema name.
     metrics:                                          ## Required line to start the list of metric items
       - downvotes:                                    ## Name of the column returned by the query. Will be appended to the metric series.
           usage: "COUNTER"                            ## Column value type.  COUNTER will make this a metric series.
           description: "Number of upvotes"            ## Helpful description of the column.

.. rubric:: Full queries-mysqld.yml example

Each column in the ``SELECT`` is named in this example, but that isn't required,
you can use a ``SELECT *`` as well. Notice the format of schema.table for the
query is included.

.. code-block:: bash

   ---
   app1_users_metrics:
     query: "SELECT app,first_name,last_name,logged_in,active_subscription,banned,upvotes,downvotes FROM app1.users"
     metrics:
       - app:
           usage: "LABEL"
           description: "Name of the Application"
       - user_type:
           usage: "LABEL"
           description: "User's privilege level within the Application"
       - first_name:
           usage: "LABEL"
           description: "User's First Name"
       - last_name:
           usage: "LABEL"
           description: "User's Last Name"
       - logged_in:
           usage: "LABEL"
           description: "User's logged in or out status"
       - active_subscription:
           usage: "LABEL"
           description: "Whether User has an active subscription or not"
       - banned:
           usage: "LABEL"
           description: "Whether user is banned or not"
       - upvotes:
           usage: "COUNTER"
           description: "Count of upvotes the User has earned. Upvotes once granted cannot be revoked, so the number can only increase."
       - downvotes:
           usage: "GAUGE"
           description: "Count of downvotes the User has earned. Downvotes can be revoked so the number can increase as well as decrease."
   ...

This custom query description should be placed in a YAML file
(``queries-mysqld.yml`` by default) on the corresponding server with |mysql|.

.. note: User is responsible for moving YAML file to the |mysql| instance
   against which the results of the custom query are to be retrieved.

In order to modify the location of the queries file, for example if you have multiple mysqld instances per server, you need to explicitly identify to the |pmm-server| |mysql| with the ``pmm-admin add`` command after the double dash::

   pmm-admin add mysql:metrics ... -- --queries-file-name=/usr/local/percona/pmm-client/query.yml

.. note: |pmm| does not control custom queries safety. User has responsibility
   for any side effects caused by the executed query on the sever and/or the
   database.

.. include:: .res/replace.txt
