# Troubleshoot upgrade issues

## PMM Server not updating correctly

If the automatic update process isn't working, you can force an update using the API:
{.power-number}

1. Open your terminal.
2. Run the update command, replacing <username>:<password> with your credentials and <pmm-server-address> with your PMM server address:
```sh
curl -X POST \
  --user <username>:<password> \
  'http://<pmm-server-address>/v1/server/updates:start' \
  -H 'Content-Type: application/json'
```
3. Wait 2-5 minutes and refresh the PMM Home page to verify the update.

## Watchtower fails with "client version is too old" error

When upgrading PMM Server via the UI, Watchtower may fail with the following error:

```
client version X.XX is too old. Minimum supported API version is X.XX, please upgrade your client to a newer version
```

This occurs when your Docker installation requires a newer API version than Watchtower supports.

**Solution**
To resolve this issue:
{.power-number}

1. Update to the latest Watchtower image:
```
docker pull percona/watchtower:latest
```
2. If the error persists, add the `DOCKER_API_VERSION` environment variable matching your Docker's minimum API version:
```
-e DOCKER_API_VERSION=1.45
```

## PMM 2.x migration issues

Starting with PMM 3.8, you must migrate through PMM 3.7.0 first. Direct migration from PMM 2.x to PMM 3.8 or later is not supported. If your migration fails, verify that you are following the correct path:
{.power-number}

1. Upgrade PMM 2.x to PMM 2.44.1
2. Migrate PMM 2.44.1 to PMM 3.7.0
3. Upgrade PMM 3.7.0 to the latest PMM version

See [Migrate PMM 2 to PMM 3](../pmm-upgrade/migrating_from_pmm_2.md) for the full procedure.

