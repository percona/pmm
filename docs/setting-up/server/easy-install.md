# Easy-install script

!!! caution alert alert-warning "Caution"
    Download and check `get-pmm.sh` before running it to make sure you know what it does.

## Linux or macOS

```sh
curl -fsSL -O https://raw.githubusercontent.com/percona/pmm/main/get-pmm.sh \
-O https://raw.githubusercontent.com/percona/pmm/main/.sha256-oneline && \
shasum -a 256 .sha256-oneline -c && \
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
