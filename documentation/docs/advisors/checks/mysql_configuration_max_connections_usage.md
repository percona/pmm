# MySQL max connection usage check

## Description

The status variable **max_used_connections** indicates the highest number of connections used since the previous system restart. This warrants attention when the value approaches the predetermined **max_connections** limit.

## Resolution

Revisit the **max_connections** configuration option and revise it inside the limit of your platform resources, if necessary. Alternatively, manage your max connection usage.

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>

