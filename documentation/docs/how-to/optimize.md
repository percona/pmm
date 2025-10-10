# Optimize

## Improving PMM Performance with Table Statistics Options

If a MySQL instance has a lot of schemas or tables, there are two options to help improve the performance of PMM when adding instances with `pmm-admin add`:

- `--disable-tablestats`, or,
- `--disable-tablestats-limit`.

!!! caution alert alert-warning "Important"
    - These settings are only for adding an instance. To change them, you must remove and re-add the instances.
    - Only one of these options can be used when adding an instance.

## Disable per-table statistics for an instance

When adding an instance with `pmm-admin add`, the `--disable-tablestats` option disables table statistics collection when there are more than the default number (1000) of tables in the instance.

### USAGE

```sh
pmm-admin add mysql --disable-tablestats
```

## Change the number of tables beyond which per-table statistics is disabled

When adding an instance with `pmm-admin add`, the `--disable-tablestats-limit` option changes the number of tables (from the default of 1000) beyond which per-table statistics collection is disabled.

### USAGE

```sh
pmm-admin add mysql --disable-tablestats-limit=<LIMIT>
```

### EXAMPLE

Add a MySQL instance, disabling per-table statistics collection when the number of tables in the instance reaches 2000.

```sh
pmm-admin add mysql --disable-tablestats-limit=2000
```
