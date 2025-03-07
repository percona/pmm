# Download and verify OVA file

This section contains guidelines on how to download and verify the OVA file.

=== "Download from the UI"

    To download an OVA file from the UI:
    {.power-number}

    1. Open a web browser.
    2. [Visit the PMM Server download page](https://www.percona.com/downloads).
    3. Choose a **Version** or use the default (the latest).
    4. Click the link for `pmm-server-{{release}}.ova` to download it. Note where your browser saves it.
    5. Right click the link for `pmm-server-{{release}}.sha256sum` and save it in the same place as the `.ova` file.
    6. (Optional) [Verify](#verify-ova-file-from-cli).


=== "Download from the CLI"

    Download the latest PMM Server OVA and checksum files:

    ```sh
    wget https://www.percona.com/downloads/pmm/{{release}}/ova/pmm-server-{{release}}.ova
    wget https://www.percona.com/downloads/pmm/{{release}}/ova/pmm-server-{{release}}.sha256sum
    ```

## Verify OVA file from CLI

Verify the checksum of the downloaded .ova file:

```sh
shasum -ca 256 pmm-server-{{release}}.sha256sum
```

