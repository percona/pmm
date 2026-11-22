# Prepare for the PostgreSQL 14 removal

PostgreSQL recently [stopped supporting version 14](https://www.postgresql.org/support/versioning/), so it no longer gets security fixes. Because of this, PMM deprecates PostgreSQL 14 in PMM 3.11.0 and will remove it completely in PMM 3.13.0 (expected January 2027).

PMM Server has been using PostgreSQL 18 for its built-in database since PMM 3.10.0. PostgreSQL 14 is only used to move your data to PostgreSQL 18 when you upgrade from PMM 3.9.1 or earlier. Once PMM 3.13.0 ships without PostgreSQL 14, PMM won't be able to upgrade a PMM Server that runs PMM 3.9.1 or earlier directly.

To avoid this, upgrade a PMM Server that runs PMM 3.9.1 or earlier to PMM 3.11.0 before PMM 3.13.0 is out.

## Check whether this affects you

This affects you only if your PMM Server runs PMM 3.9.1 or earlier with the built-in database, including a PMM Server that you migrated from PMM 2. It doesn't affect you if your PMM Server already runs PMM 3.10.0 or later, if you run a [PMM HA cluster](upgrade_helm_ha.md), or if you use an [external PostgreSQL database](#external-postgresql-databases).

If you're not sure, check which PostgreSQL data folders your PMM Server has:

=== "Docker"

    ```sh
    docker exec pmm-server ls /srv
    ```

=== "Podman and AWS"

    ```sh
    podman exec pmm-server ls /srv
    ```

=== "Helm"

    ```sh
    kubectl exec <pod> --namespace <namespace> -- ls /srv
    ```

- `postgres18` in the list: your data is already on PostgreSQL 18. You don't need to do anything.
- Only `postgres14` in the list: upgrade to PMM 3.11.0, as described below.

## Upgrade to PMM 3.11.0

Upgrade PMM Server as usual. PMM 3.11.0 moves your data to PostgreSQL 18 automatically:
{.power-number}

1. Upgrade PMM Server with the method you use:

    - [Docker](upgrade_docker.md)
    - [Podman](upgrade_podman.md)
    - [AWS](upgrade_aws.md)
    - [Helm](upgrade_helm.md)

2. Wait until PMM Server is running. The first start takes longer than usual, because PMM Server moves your data to PostgreSQL 18 before it starts its services.

3. Check that the move finished. The log shows `PostgreSQL 14 to 18 migration completed successfully.`:

    === "Docker"

        ```sh
        docker logs pmm-server 2>&1 | grep "migration completed"
        ```

    === "Podman and AWS"

        ```sh
        podman logs pmm-server 2>&1 | grep "migration completed"
        ```

    === "Helm"

        ```sh
        kubectl logs <pod> --namespace <namespace> | grep "migration completed"
        ```

    To double-check, run the [folder check](#check-whether-this-affects-you) again. `/srv` now has a `postgres18` folder, and `postgres14` is renamed to `postgres14.old`.

## If you upgrade after PMM 3.13.0 is out

Once PMM 3.13.0 is out, a PMM Server that still runs PMM 3.9.1 or earlier will need to upgrade in two steps:
{.power-number}

1. Upgrade to a version from PMM 3.10.0 through 3.12.x, so that PMM Server moves your data to PostgreSQL 18.
2. Upgrade to PMM 3.13.0 or later.

For step 1, always name the version. These install the latest release, so after PMM 3.13.0 is out, they would skip step 1:

- the `percona/pmm-server:3` image tag
- the easy-install script without `--tag`
- the **Upgrade** button in PMM 3.8.1 or earlier, and `POST /v1/server/updates:start` without `new_image`

## External PostgreSQL databases

If you [use an external PostgreSQL database](../reference/third-party/postgresql.md) instead of the built-in one, PMM Server requires PostgreSQL 15 or later starting with PMM 3.13.0. If your external database runs an earlier version, upgrade it before you upgrade PMM Server to PMM 3.13.0.
