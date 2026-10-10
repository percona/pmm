# MySQL accounts accessible from public networks

The `mysql_private_networks_only` check warns about MySQL accounts that can connect from public networks.

## Description

The host part of a MySQL account name defines the hosts from which the account can connect. An account that accepts connections from public IP addresses is exposed to password-guessing attacks from the internet.

The check reads the `mysql.user` table and lists every account that can connect from a public address. An account passes the check if its host part is `localhost`, or if every address that the host part matches lies in one of the following private ranges:

- IPv4 private networks: `10.0.0.0/8`, `172.16.0.0/12` and `192.168.0.0/16`
- Loopback and link-local addresses: `127.0.0.0/8`, `169.254.0.0/16`, `::1`, `fe80::/10`
- Other reserved ranges: `100.64.0.0/10`, `192.0.0.0/24`, `198.18.0.0/15`, `224.0.0.0/24` and `fc00::/7`

The check evaluates the following forms of the host part:

- A single IPv4 or IPv6 address, such as `10.0.1.15`
- An IPv4 network in CIDR or netmask notation, such as `10.0.0.0/8` or `10.0.0.0/255.0.0.0`
- An IPv4 pattern whose last octet is the `%` wildcard, such as `192.168.%`

A network or a pattern passes only if it lies fully inside one private range, so the check reports a host part such as `0.0.0.0/0`. The check also reports host names and other wildcard patterns, such as `%`, because it can't verify that they match only private addresses.

## Resolution

Restrict each reported account to the hosts that need access:

- To change the host part of an account, use `RENAME USER`, for example: `RENAME USER 'app'@'%' TO 'app'@'10.0.1.15';`
- If an account isn't needed any more, remove it with `DROP USER`.

For the forms that the host part can take, see [Specifying Account Names](https://dev.mysql.com/doc/refman/8.4/en/account-names.html) in the MySQL documentation.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
