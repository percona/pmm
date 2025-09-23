# Unsupported MySQL version

## Description

This check verifies the current MySQL versions and identifies if it is unsupported.

An unsupported MySQL version in production can lead to security vulnerabilities, bugs, and instability issues. Also there will not be any support available by the vendors for any identified issues for the fixtures.

## Resolution

We do not support an upgrade from 5.6 directly to 8.0. You should first upgrade to the latest version of 5.6 and then [follow the steps to upgrade to 5.7](https://docs.percona.com/percona-server/5.7/upgrade.html).

You can then [upgrade from 5.7 to 8.0](https://docs.percona.com/percona-server/8.0/upgrading_guide.html).

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
