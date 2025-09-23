# Verify if local infile global variable is disabled

## Description

The **LOAD DATA** statement loads a data file into a table. The statement can load a file located on the server host, or, if the LOCAL keyword is specified, on the client host.

The LOCAL version of **LOAD DATA** has two potential security issues:

- Because LOAD DATA LOCAL is an SQL statement, parsing occurs on the server side.  The transfer of file from the client host to the server host is initiated by the MySQL server, which tells the client the file named in the statement. 
  
  In theory, a patched server could tell the client program to transfer a file of the server's choosing rather than the file named in the statement. Such a server could access any file on the client host to which the client user has read access. 
  
  A patched server could in fact reply with a file-transfer request to any statement, not just LOAD DATA LOCAL, so a more fundamental issue is that clients should not connect to untrusted servers.

- In a Web environment where the clients are connecting from a Web server, a user could use LOAD DATA LOCAL to read any files that the Web server process has read access to (assuming that a user could run any statement against the SQL server). 
  
  In this environment, the client with respect to the MySQL server actually is the Web server, not a remote program being run by users who connect to the Web server.

To avoid connecting to untrusted servers, clients can establish a secure connection and verify the server identity by connecting using the **--ssl-mode=VERIFY_IDENTITY** option and the appropriate CA certificate.

To avoid LOAD DATA issues, clients should avoid using LOCAL unless proper client-side precautions have been taken.

## Resolution

Administrators and applications can configure whether to permit local data loading as follows:

On the server side:
- The local_infile system variable controls server-side LOCAL capability. Depending on the local_infile setting, the server refuses or permits local data loading by clients that request local data loading.
  
- By default, local_infile is disabled. (This is a change from previous versions of MySQL.) To cause the server to refuse or permit LOAD DATA LOCAL statements explicitly (regardless of how client programs and libraries are configured at build time or runtime), start mysqld with local_infile disabled or enabled. local_infile can also be set at runtime.

## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }
