# MongoDB authentication is disabled

The `mongodb_auth` check warns if authentication is disabled on a MongoDB instance.

## Description

Without authentication, any client that can reach the instance over the network can connect to it, read and change data, and run administrative commands.

The check reads the startup options of the instance with the `getCmdLineOpts` command. It considers authentication enabled if one of these options is set:

- `security.authorization` set to `enabled`
- `security.keyFile`
- `security.clusterAuthMode`

The last two options enable internal authentication between the members of a replica set or a sharded cluster, which also enforces access control for clients. If none of these options is set, the check returns a warning.

## Resolution

To enable access control on a standalone instance:
{.power-number}

1. Connect to the instance with `mongosh` and create a user administrator in the `admin` database, if you don't have one yet.
2. Edit the `mongod.conf` file and turn on authorization:

    ```yaml
    security:
      authorization: enabled
    ```

3. Restart `mongod` to apply the change.
4. Connect as the user administrator and create users with the roles that your applications need.

To set up users and roles in more detail, follow the MongoDB guide to [enable access control on a self-managed deployment](https://www.mongodb.com/docs/manual/tutorial/enable-authentication/).

For a replica set or a sharded cluster, enable internal authentication with a keyfile or X.509 certificates instead. To avoid downtime, [enforce keyfile access control on the running replica set](https://www.mongodb.com/docs/manual/tutorial/enforce-keyfile-access-control-in-existing-replica-set-without-downtime/).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
