# Easy-install script

!!! caution alert alert-warning "Caution"
    - This is a [technical preview] and is subject to change.
    - Download and check `get-pmm2.sh` before running it to make sure you know what it does.

## Linux

```sh
curl -fsSL -O https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -O https://raw.githubusercontent.com/percona/pmm/PMM-2.0/.sha256-oneline && \
sha256sum .sha256-oneline -c && \
chmod +x ./get-pmm.sh && \
./get-pmm.sh
```

## MacOS

```sh
curl -fsSL -O https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -O https://raw.githubusercontent.com/percona/pmm/PMM-2.0/.sha256-oneline && \
shasum .sha256-oneline -c && \
chmod +x ./get-pmm.sh && \
./get-pmm.sh
```

These commands:

- Download the script;
- Check its integrity;
- Make the script executable;
- Run it. The script will:
    - install Docker if not already installed;
    - if there is a PMM Server docker container running, stop it and back it up;
    - pull and run the latest PMM Server docker image.

[technical preview]: ../../details/glossary.md#technical-preview
