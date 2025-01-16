# Disable per-table statistics

You can optimize PMM by disabling per-table statistics for an instance as follows:

When adding an instance with `pmm-admin add`, the `--disable-tablestats` option disables table statistics collection when there are more than the default number (1000) of tables in the instance.

## USAGE

```sh
pmm-admin add mysql --disable-tablestats
```