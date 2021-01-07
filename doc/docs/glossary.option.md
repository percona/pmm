# PMM Server Additional Options

This glossary contains the additional parameters that you may pass when starting PMM Server.

## Passing options to PMM Server docker at first run

Docker allows configuration options to be passed using the flag `-e` followed by the option you would like to set.

Here, we pass more than one option to PMM Server along with the **docker run** command. Run this command as root or by using the **sudo** command.

```
$ docker run -d -p 80:80 \
  --volumes-from pmm-data \
  --name pmm-server \
  -e SERVER_USER=jsmith \
  -e SERVER_PASSWORD=SomeR4ndom-Pa$$w0rd \
  --restart always \
  percona/pmm-server:1
```

## Passing options to *PMM Server* for an already deployed docker instance

Docker doesn’t support changing environment variables on an already provisioned container, therefore you need to stop the current container and start a new container with the new options. variable with **docker start** if you want to change the setting for existing installation, because **docker start** cares to keep container immutable and doesn’t support changing environment variables. Therefore if you want container with different properties,  you should run a new container instead.

1. Stop and Rename the old container:

    ```
    docker stop pmm-server
    docker rename pmm-server pmm-server-old
    ```

2. Ensure you are running the latest version of PMM Server:

    ```
    docker pull percona/pmm-server:latest
    ```

    **WARNING**: When you destroy and recreate the container, all the updates you have done through PMM Web interface will be lost. What’s more, the software version will be reset to the one in the Docker image. Running an old PMM version with a data volume modified by a new PMM version may cause unpredictable results. This could include data loss.

3. Start the container with the new settings. For example, changing [`METRICS_RESOLUTION`](#metrics-resolution) would look as follows:

    ```
    docker run -d \
      -p 80:80 \
      --volumes-from pmm-data \
      --name pmm-server \
      --restart always \
      -e METRICS_RESOLUTION=5s \
      percona/pmm-server:latest
    ```

4. Once you’re satisfied with the new container deployment options and you don’t plan to revert, you can remove the old container:

    ```
    docker rm pmm-server-old
    ```

## List of PMM Server Parameters

### DISABLE_TELEMETRY

With telemetry enabled, your PMM Server sends some statistics to [v.percona.com](http://v.percona.com) every 24 hours. This statistics includes the following details:

* PMM Server unique ID
* PMM version
* The name and version of the operating system
* MySQL version
* Perl version

If you do not want your PMM Server to send this information, disable telemetry when running your Docker container:

```
$ docker run ... -e DISABLE_TELEMETRY=true ... percona/pmm-server:1
```

### METRICS_RETENTION

This option determines how long metrics are stored at PMM Server. The value is passed as a combination of hours, minutes, and seconds, such as **720h0m0s**. The minutes (a number followed by *m*) and seconds (a number followed by *s*) are optional.

To set the `METRICS_RETENTION` option to 8 days, set this option to *192h*.

Run this command as root or by using the **sudo** command

```
$ docker run ... -e METRICS_RETENTION=192h ... percona/pmm-server:1
```

### QUERIES_RETENTION

This option determines how many days queries are stored at PMM Server.

```
$ docker run ... -e QUERIES_RETENTION=30 ... percona/pmm-server:1
```

### ORCHESTRATOR_ENABLED

This option enables Orchestrator (See [Orchestrator](architecture.md#pmm-using-orchestrator)). By default it is disabled. It is also disabled if this option contains **false**.

```
$ docker run ... -e ORCHESTRATOR_ENABLED=true ... percona/pmm-server:1
```

### ORCHESTRATOR_USER

Pass this option, when running your PMM Server via Docker to set the orchestrator user. You only need this parameter (along with `ORCHESTRATOR_PASSWORD` if you have set up a custom Orchestrator user.

This option has no effect if the `ORCHESTRATOR_ENABLED` option is set to **false**.

```
$ docker run ... -e ORCHESTRATOR_ENABLED=true ORCHESTRATOR_USER=name -e ORCHESTRATOR_PASSWORD=pass ... percona/pmm-server:1
```

### ORCHESTRATOR_PASSWORD

Pass this option, when running your PMM Server via Docker to set the orchestrator password.

This option has no effect if the `ORCHESTRATOR_ENABLED` option is set to **false**.

```
$ docker run ... -e ORCHESTRATOR_ENABLED=true ORCHESTRATOR_USER=name -e ORCHESTRATOR_PASSWORD=pass ... percona/pmm-server:1
```

### SERVER_USER

By default, the user name is `pmm`. Use this option to use another user name.

Run this command as root or by using the **sudo** command.

```
$ docker run ... -e SERVER_USER=USER_NAME ... percona/pmm-server:1
```

### SERVER_PASSWORD

Set the password to access the PMM Server web interface.

Run this command as root or by using the **sudo** command.

```
$ docker run ... -e SERVER_PASSWORD=YOUR_PASSWORD ... percona/pmm-server:1
```

By default, the user name is `pmm`. You can change it by passing the `SERVER_USER` variable.

### METRICS_RESOLUTION

This environment variable sets the minimum resolution for checking metrics. You should set it if the latency is higher than 1 second.

Run this command as root or by using the **sudo** command.

```
$ docker run ... -e METRICS_RESOLUTION=VALUE ... percona/pmm-server:1
```

### METRICS_MEMORY

By default, Prometheus in PMM Server uses all available memory for storing the most recently used data chunks.  Depending on the amount of data coming into Prometheus, you may require to allow less memory consumption if it is needed for other processes.

If you are still using a version of PMM prior to 1.13 you might need to set the metrics memory by passing the `METRICS_MEMORY` environment variable along with the **docker run** command.

Run this command as root or by using the **sudo** command. The value must be passed in kilobytes. For example, to set the limit to 4 GB of memory run the following command:

```
$ docker run ... -e METRICS_MEMORY=4194304 ... percona/pmm-server:1
```

### DISABLE_UPDATES

To update your PMM from web interface you only need to click the Update on the home page. The `DISABLE_UPDATES` option is useful if updating is not desirable. Set it to **true** when running PMM in the Docker container.

Run this command as root or by using the **sudo** command.

```
$ docker run ... -e DISABLE_UPDATES=true ... percona/pmm-server:1
```

The `DISABLE_UPDATES` option removes the Update button from the interface and prevents the system from being updated manually.
