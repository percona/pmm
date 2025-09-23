# User with open to the word scope

## Description

For more information, see [Specifying Account Names](https://dev.mysql.com/doc/refman/8.0/en/account-names.html ).

The host name part of an account name can take many forms, and wildcards are permitted:
- Because IP wildcard values are permitted in host values (for example, '198.51.100.%' to match every host on a subnet), someone could try to exploit this capability by naming a host 198.51.100.somewhere.com. To foil such attempts, MySQL does not perform matching on host names that start with digits and a dot. For example, if a host is named 1.2.example.com, its name never matches the host part of account names. An IP wildcard value can match only IP addresses, not host names.
- For a host value specified as an IPv4 address, a netmask can be given to indicate how many address bits to use for the network number. Netmask notation cannot be used for IPv6 addresses.


## Resolution

Remove any user that does not have a name in the mysql.user table. Or change the host to something with a limited scope.
 
## Need more support from Percona?

Subscribe to Percona Platform to get database support with guaranteed SLAs or proactive database management services from the Percona team.

[Learn more :fontawesome-solid-paper-plane:](https://per.co.na/subscribe){ .md-button }