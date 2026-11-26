# PostgreSQL users with superuser privileges

The `postgresql_super_role` check warns if PostgreSQL roles that can log in have superuser privileges.

## Description

A superuser bypasses all permission checks, except the right to log in. A leaked superuser password, or a mistake made in a superuser session, can affect every database on the server.

The check reads the `pg_roles` catalog and lists the roles that can log in and have either:

- The `SUPERUSER` attribute
- Membership in the `rds_superuser` role, on Amazon RDS

The check skips roles whose names start with `pg_`, and the `postgres`, `rdsadmin` and `pmm_user` roles. If it finds any superuser roles, the check returns a warning that lists them.

## Resolution

For each reported role, verify whether it needs superuser privileges. If it doesn't, remove them:

- To remove the `SUPERUSER` attribute, run `ALTER ROLE app_user NOSUPERUSER;`
- On Amazon RDS, revoke the membership: `REVOKE rds_superuser FROM app_user;`

Then grant the role only the privileges that it needs. For common tasks, such as monitoring or reading all data, PostgreSQL provides predefined roles like `pg_monitor` and `pg_read_all_data`. To choose the right privileges, compare the PostgreSQL [role attributes](https://www.postgresql.org/docs/current/role-attributes.html) and [predefined roles](https://www.postgresql.org/docs/current/predefined-roles.html).

## Need more support from Percona?

Percona experts bring years of experience in tackling tough database performance issues and design challenges.

<div data-tf-live="01JKGYABNVYHQ8A91QNW69A9TP"></div><script src="//embed.typeform.com/next/embed.js"></script>
