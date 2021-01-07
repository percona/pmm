# *MySQL MyRocks Metrics* Dashboard

The [MyRocks](http://myrocks.io) storage engine developed by Facebook based on the RocksDB
storage engine is applicable to systems which primarily interact with the
database by writing data to it rather than reading from it. RocksDB also
features a good level of compression, higher than that of the InnoDB storage
engine, which makes it especially valuable when optimizing the usage of hard
drives.

PMM collects statistics on the MyRocks storage engine for MySQL in the
Metrics Monitor information for this dashboard comes from the
*Information Schema* tables.

### Metrics

<!-- -*- mode: rst -*- -->
<!-- Tips (tip) -->
<!-- Abbreviations (abbr) -->
<!-- Docker commands (docker) -->
<!-- Graphical interface elements (gui) -->
<!-- Options and parameters (opt) -->
<!-- pmm-admin commands (pmm-admin) -->
<!-- SQL commands (sql) -->
<!-- PMM Dashboards (dbd) -->
<!-- * Text labels -->
<!-- Special headings (h) -->
<!-- Status labels (status) -->
