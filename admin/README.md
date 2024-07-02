# pmm-admin

pmm-admin for PMM 2.x.

# Contributing notes

## Pre-requirements:
git, make, curl, go, gcc, pmm-server, pmm-agent

## Local setup
### To run pmm-admin commands
- Run [pmm-server docker container](https://hub.docker.com/r/percona/pmm-server) or [pmm-managed](https://github.com/percona/pmm-managed).  
- Run pmm-agent `cd ../agent`.
- Run pmm-admin commands.
    ```shell script
    go run main.go status
    ```

You should see something like this
 ```shell script
Agent ID: fcbe3cb4-a95a-43f4-aef5-c3494caa5132
Node ID : 77be6b4d-a1d9-4687-8fae-7acbaee7db47

PMM Server:
        URL    : https://127.0.0.1:443/
        Version: 2.2.0-HEAD-fcde194

PMM-agent:
        Connected : true
        Time drift: 41.93µs
        Latency   : 211.026µs

Agents:
        3329a405-8a5d-4414-9890-b6ae4209e0cc NODE_EXPORTER RUNNING
```
It means that everything works.

## Testing
pmm-admin doesn't require setting-up environment.  
Run `make test` to run tests. 
