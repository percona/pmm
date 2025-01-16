# Improve PMM performance

You can improve PMM performance with the Table Statistics Options as follows:

If a MySQL instance has a lot of schemas or tables, there are two options to help improve the performance of PMM when adding instances with `pmm-admin add`:

- `--disable-tablestats`, or,
- `--disable-tablestats-limit`.

!!! caution alert alert-warning "Important"
    - These settings are only for adding an instance. To change them, you must remove and re-add the instances.
    - Only one of these options can be used when adding an instance.
