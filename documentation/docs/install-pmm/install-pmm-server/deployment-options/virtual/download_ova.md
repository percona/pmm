# Download and verify OVA file


!!! warning "OVF support ending in PMM 3.9.0"
    OVF/OVA distribution is deprecated starting with PMM 3.7.0 and will be removed in PMM 3.9.0 (expected July 2026). No new OVA images will be published after that release. Migrate to [Docker](../docker/index.md) or another supported deployment method before PMM 3.9.0.
Download the Virtual Appliance (OVA) file to deploy PMM Server as a virtual machine.

## Supported platforms

The PMM Server OVA works with:

- Oracle VirtualBox
- Other OVF-compatible virtualization platforms

## Download options

=== "Download from the UI"

    To download an OVA file from the UI:
    {.power-number}

    1. Visit the [Percona Downloads page](https://www.percona.com/downloads) from a web browser.
    2. Make sure PMM 3 is selected, then choose a PMM version and select **SERVER - VIRTUAL APPLIANCE OVF**.
    3. Click the **DOWNLOAD** link for `pmm-server-{{release}}.ova` and note where your browser saves it.
    4. Right-click the link for `pmm-server-{{release}}.sha256sum` and save it in the same place as the `.ova` file.

=== "Download with CLI"

    Download the latest PMM Server OVA and checksum files:

    ```sh
    # Download the OVA file (replace X.Y.Z with the desired version)
    wget https://downloads.percona.com/downloads/pmm/X.Y.Z/ova/PMM-Server-X.Y.Z.ova
    
    # Download the checksum file
    wget https://downloads.percona.com/downloads/pmm/X.Y.Z/ova/PMM-Server-X.Y.Z.ova.sha256sum
    ```


## Verify OVA integrity

After downloading, verify the file integrity to ensure it hasn't been corrupted:

```sh
# Navigate to the download location
cd /path/to/download

# Verify the checksum
sha256sum -c PMM-Server-X.Y.Z.ova.sha256sum
```

You should see output confirming the file is OK:
`PMM-Server-X.Y.Z.ova: OK`

## Next steps
After downloading the OVA file, [Deploy on VirtualBox](../virtual/virtualbox.md).