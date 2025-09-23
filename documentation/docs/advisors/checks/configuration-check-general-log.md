# MySQL General log is active

## Description

The general query log contains the following information:

* Every time a client connects or disconnects

* Every SQL statement received from the clients

Enabling the general log can seriously impact disk space and overall performance. By default, the general query log is disabled.

## Resolution

Disable the general query log in the configuration file, and restart the instance for the change to take effect.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
