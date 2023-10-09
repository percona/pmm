## MongoDB replica set with PBM and PMM Agent

This directory contains docker compose and scripts for Backups Dev environment bootstrap. This environment is based
on MongoDB replica set running in docker, as a result, it only supports logical backups/restores.

### Usage

Note: If your already have running PMM Server that wasn't started from this repo (devcontainer), then you
need to specify its container name and network with env variables `PMM_SERVER_HOST` and `PMM_NETWORK`.
You can check network name with this
command `docker inspect <container name> -f '{{range $k, $v := .NetworkSettings.Networks}}{{printf "%s\n" $k}}{{end}}'`.
Replace `<container name>` with your pmm server container name.

```shell
    # export PMM_SERVER_HOST=pmm-server
    # export PMM_NETWORK=pmm_default
    
    make bootstrap
```

This command will build custom Docker image with MongoDB, PBM and PMM agents inside. All tools will be preconfigured,
PMM agents will be
registered on PMM server as well as MongoDB instances.

To shutdown env just invoke `make down`