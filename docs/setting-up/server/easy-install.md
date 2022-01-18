# Easy-install script

!!! caution alert alert-warning "Caution"
    You can download and check `get-pmm.sh` before running it from our [github]:

## Linux or macOS

```sh
curl -fsSL https://www.percona.com/get/pmm | /bin/bash
```

This script:

- Installs Docker if it is not installed.
- If a PMM Server Docker container is running, it is stopped and backed up.
- Pulls and runs the latest PMM Server Docker image.


[github]: https://github.com/percona/pmm/blob/main/get-pmm.sh
