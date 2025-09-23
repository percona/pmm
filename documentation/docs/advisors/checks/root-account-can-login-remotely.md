# Root account can log in remotely

## Description

The Root user is a high-privilege user that can perform any kind of operation on the system.

It is best practice to limit the access to this specific user only when connecting from local instances, 
and to eventually create another user with the specific DBA privileges, that will be able to connect from remote.

## Resolution

Remove any root user that is not having ‘127.0.0.1’ or ‘localhost’ as host definition. Create a DBA user with the required privileges and specific for the schema that the DBA needs to handle.  

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }