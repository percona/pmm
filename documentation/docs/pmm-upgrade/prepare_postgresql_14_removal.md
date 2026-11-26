# Prepare for the PostgreSQL 14 removal

PostgreSQL recently [stopped supporting version 14](https://www.postgresql.org/support/versioning/), so it no longer gets security fixes. PMM already switched the built-in database of PMM Server to PostgreSQL 18 in PMM 3.10.0 and includes PostgreSQL 14 only to migrate existing data when upgrading from PMM 3.9.1 or earlier.

Because of this, PMM deprecates PostgreSQL 14 in PMM 3.11.0 and will remove it completely in PMM 3.13.0 (expected January 2027).

Once PMM 3.13.0 removes PostgreSQL 14, PMM can no longer migrate that data, so it can't upgrade a PMM Server that runs PMM 3.9.1 or earlier directly. If your PMM Server runs PMM 3.9.1 or earlier:

- before PMM 3.13.0: upgrade to PMM 3.11.0 or 3.12.x as usual. Your data moves to PostgreSQL 18 automatically.
- after PMM 3.13.0: [upgrade in two steps](#if-you-upgrade-after-pmm-3130-is-out). First upgrade to a version from PMM 3.10.0 through 3.12.x, then to PMM 3.13.0 or later.

## Check whether this affects you

What to check depends on which PostgreSQL database PMM Server uses: the built-in one that comes with the `pmm-server` container, or an external one that you configured instead.

### Built-in database

The PostgreSQL 14 removal affects a PMM Server that runs PMM 3.9.1 or earlier with the built-in database because upgrading it migrates its data from PostgreSQL 14 to PostgreSQL 18. It doesn't apply if your PMM Server already runs PMM 3.10.0 or later, or if you run a [PMM HA cluster](upgrade_helm_ha.md).

If you're not sure, check which PostgreSQL data folders your PMM Server has:

=== "Docker"

    ```sh
    docker exec pmm-server ls /srv
    ```

=== "Podman and AWS"

    ```sh
    podman exec pmm-server ls /srv
    ```

=== "Helm (single instance)"

    ```sh
    kubectl exec <pod> --namespace <namespace> -- ls /srv
    ```

The folders that the command lists show whether your data is already on PostgreSQL 18:

- `postgres18` folder: your data is already on PostgreSQL 18, so you don't need to do anything.
- `postgres14` folder but no `postgres18` folder: your data is still on PostgreSQL 14. Before PMM 3.13.0 is out, [upgrade directly to PMM 3.11.0 or 3.12.x](#upgrade-to-pmm-3110). After it's out, [upgrade in two steps](#if-you-upgrade-after-pmm-3130-is-out).
- no `postgres14` or `postgres18` folder: your PMM Server doesn't use the built-in database, which means it runs in PMM HA mode or uses an external database. With PMM HA, you don't need to do anything. With an external database, [make sure it runs PostgreSQL 15 or later](#external-database) before you upgrade to PMM 3.13.0.

### External database

PMM 3.13.0 requires PostgreSQL 15 or later for external databases. This affects you if you [configured PMM Server to use an external PostgreSQL database](../reference/third-party/postgresql.md) instead of the built-in one, and it runs an earlier version.

To check the version, run `SELECT version();` on the external database. If it's earlier than PostgreSQL 15, upgrade it before you upgrade PMM Server to PMM 3.13.0.

If you need help upgrading your external database, contact [Percona Support](https://www.percona.com/services/support).

## Upgrade to PMM 3.11.0

Upgrade PMM Server as usual. PMM 3.11.0 migrates your data to PostgreSQL 18 automatically:
{.power-number}

1. Upgrade PMM Server with the method you use:

    - [Docker](upgrade_docker.md)
    - [Podman](upgrade_podman.md)
    - [AWS](upgrade_aws.md)
    - [Helm](upgrade_helm.md)

2. Wait until PMM Server is running. The first start takes longer than usual, because PMM Server migrates your data to PostgreSQL 18 before it starts its services.

3. Check that the migration finished. The log shows `PostgreSQL 14 to 18 migration completed successfully.`:

    === "Docker"

        ```sh
        docker logs pmm-server 2>&1 | grep "migration completed"
        ```

    === "Podman and AWS"

        ```sh
        podman logs pmm-server 2>&1 | grep "migration completed"
        ```

    === "Helm (single instance)"

        ```sh
        kubectl logs <pod> --namespace <namespace> | grep "migration completed"
        ```

    To double-check, run the [folder check](#check-whether-this-affects-you) again. `/srv` now has a `postgres18` folder, and `postgres14` is renamed to `postgres14.old`.

## If you upgrade after PMM 3.13.0 is out

Once PMM 3.13.0 is out, a PMM Server that still runs PMM 3.9.1 or earlier needs an intermediate version, so that PMM migrates its data to PostgreSQL 18 first:

PMM 3.9.1 or earlier > PMM 3.10.0 through 3.12.x (intermediate version) > PMM 3.13.0 or later

To upgrade in two steps:
{.power-number}

1. Upgrade to the intermediate version, for example PMM 3.11.0, by naming its version:

    - Docker: `percona/pmm-server:3.11.0` instead of `percona/pmm-server:3`
    - Helm: `--set image.tag=3.11.0`
    - Automated script: `--tag 3.11.0`

    In PMM 3.8.1 or earlier, don't use the **Upgrade** button, because it installs the latest release.

    Then [check that the migration finished](#upgrade-to-pmm-3110).

2. Upgrade to PMM 3.13.0 or later, with the [method you use](index.md#available-upgrade-methods).
