# Easy-install PMM Server

The following steps will guide you through the installation of PMM Server:
{.power-number}

1. Download and run the PMM easy installation script from [github](https://github.com/percona/pmm/blob/main/get-pmm.sh). The install script only runs on Linux-compatible systems. To use it, run the command with `sudo` privileges or as root.

    ??? note "What's happening under the hood"
        - Installs Docker if it is not installed on your system.
        - Stops and renames any currently running PMM Server Docker container from `pmm-server` to `pmm-server-{timestamp}`. This old pmm-server container is not a recoverable backup.
        - Pulls and runs the latest PMM Server Docker image.
        - To run PMM in the `Interactive` mode, execute the following command:

            ```sh
            curl -fsSLO https://www.percona.com/get/pmm (or wget https://www.percona.com/get/pmm)
            chmod +x pmm
            ./pmm --interactive
            ```

2. Install PMM Server using `cURL` or `wget`:

    === "cURL"

        ```sh
        curl -fsSL https://www.percona.com/get/pmm | /bin/bash
        ```

    === "wget"

        ```sh
        wget -qO - https://www.percona.com/get/pmm | /bin/bash    
        ```

3. Log in to PMM with the default login credentials that are provided after the installation is completed.

## Next step: Set up PMM Client

Now that you have PMM Server set up we need to go to your databases and add PMM Client so that PMM Server can communicate with your databases. Learn how in the button below.

[Set up PMM Client :material-arrow-right:](../set-up-pmm-client/index.md){ .md-button .md-button--primary }

If you want to try something else before anything, here are some other ideas for next steps:

- [Backup](#) PMM Server and its data
- [Update](#) PMM Server
- [Restore](#) PMM Server
- [Remove](#) PMM Server
- [Use Environment Variables](#) to set PMM Server parameters.