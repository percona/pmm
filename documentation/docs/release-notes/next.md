# Percona Monitoring and Management {{VERSION}}

**Release date**: TBD

Percona Monitoring and Management (PMM) is an open source solution for monitoring and managing MySQL, PostgreSQL, MongoDB, Valkey, and Redis databases, on premises or in the cloud.

## 📋 Release summary

## ✨ Release highlights

## 🔒 Security updates

- [PMM-15194](https://perconadev.atlassian.net/browse/PMM-15194): PMM now encrypts every credential it stores in a new format, including Valkey TLS certificates and keys and the S3 credentials of backup locations, which earlier versions stored unencrypted. Each credential is bound to the field it is stored in, so a value copied into another field doesn't decrypt. The first start after the upgrade re-encrypts the stored credentials with your existing key, and keeps the values it rewrites in a backup file. It also repairs credentials that key rotation corrupted before PMM 3.9.1, which the [PMM 3.9.1 release notes](3.9.1.md) asked you to re-add, using the previous key that the rotation left next to the key file. PMM Server no longer generates a new key when its key file is missing or doesn't match the stored credentials: it refuses to start instead. Back up the key file with your PMM data before you upgrade, because you can't downgrade afterward. In high availability mode, stop every PMM Server node before the upgrade, also with the PMM HA Helm chart. For details, see [Upgrade from PMM 3.10.0 or earlier](../admin/security/data_encryption.md#upgrade-from-pmm-3100-or-earlier).

## 📈 Improvements

## ✅ Fixed issues

## 🔶 Known issues

## 🚀 Ready to upgrade to PMM {{VERSION}}?

- [New installation](../quickstart/quickstart.md)
- [Upgrading from PMM 3](../pmm-upgrade/index.md)
- [Upgrading from PMM 2](../pmm-upgrade/migrating_from_pmm_2.md)
