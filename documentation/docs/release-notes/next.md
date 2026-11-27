# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

## 📈 Improvements

- [PMM-15899](https://perconadev.atlassian.net/browse/PMM-15899): PMM Server now also keeps the VictoriaMetrics password private. It copies the password from `PMM_VM_URL` to a file that only it can read, so the password is no longer in its service environments, configuration files or command lines. To also hide it from the container settings, for example in `docker inspect`, you can now [provide it in a file](../admin/security/credentials_as_files.md) instead, as with the other credentials since PMM 3.11.0.

    To make sure that PMM Server starts after the upgrade, [check `PMM_VM_URL` before you upgrade](../admin/security/credentials_as_files.md#check-your-environment-variables-before-you-upgrade).

## ✅ Fixed issues

- [PMM-15899](https://perconadev.atlassian.net/browse/PMM-15899): Fixed an issue where PMM Server never became ready when the VictoriaMetrics username or password in `PMM_VM_URL` contained `%` or a character that PMM Server percent-encodes, such as `:`, `@`, `!` or `'`.

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
