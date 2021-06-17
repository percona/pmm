# Easy-install script

!!! caution alert alert-warning "Caution"
    - This is a [technical preview] and is subject to change.
    - Download and check `get-pmm2.sh` before running it to make sure you know what it does.

```sh
curl -fsSL https://raw.githubusercontent.com/percona/pmm/PMM-2.0/get-pmm.sh -o get-pmm2.sh && \
chmod +x get-pmm2.sh && \
./get-pmm2.sh
```

These commands:

- install Docker if not already installed;
- if there is a PMM Server docker container running, stop it and back it up;
- pull and run the latest PMM Server docker image.

[technical preview]: ../../details/glossary.md#technical-preview