# Configuration issues

This section focuses on configuration issues, such as PMM-agent connection, adding and removing services for monitoring, and so on.

## Client-Server connections

There are many causes of broken network connectivity.

The container is constrained by the host-level routing and firewall rules when using [using Docker](../install-pmm/install-pmm-server/index.md). For example, your hosting provider might have default `iptables` rules on their hosts that block communication between PMM Server and PMM Client, resulting in *DOWN* targets in VictoriaMetrics. If this happens, check the firewall and routing settings on the Docker host.

PMM can also generate diagnostics data that can be examined and/or shared with our support team to help solve an issue. You can get collected logs from PMM Client using the pmm-admin summary command.

Logs obtained in this way include PMM Client logs and logs received from the PMM Server, and stored separately in the `client` and `server` folders. The `server` folder also contains its `client` subfolder with the self-monitoring client information collected on the PMM Server.

For additional debugging information, use the `--pprof` flag to include [pprof](https://github.com/google/pprof) debug profiles: `pmm-admin summary --pprof`.

You can get PMM Server logs with either of these methods:

**Direct download**

In a browser, visit `https://<address-of-your-pmm-server>/logs.zip`.

This downloads a bundle containing PMM Server and PMM Client log files from `/srv/logs/` inside the container. 

To control how many lines are included, add the `line-count` parameter to the URL:

- specific line count: `https://<address-of-your-pmm-server>/logs.zip?line-count=10000`
- full logs (all lines): `https://<address-of-your-pmm-server>/logs.zip?line-count=-1`

**From Help menu**

To obtain the logs:
{.power-number}

1. From the main menu, choose **Help > PMM Logs > Export logs**.

2. Save the downloaded file to share with our support team if needed.

### Monitoring stops while pmm-agent is running

A PMM Client node can stop sending metrics and Query Analytics data while `pmm-agent` and its exporters keep running. Inventory operations for the services on that node fail with `pmm-agent with ID <AGENT_ID> is not currently connected`.

Starting with PMM 3.10.0, `pmm-agent` recovers from this state without a restart:

- It keeps answering the connection checks of PMM Server while it applies a configuration change, so a slow change doesn't disconnect the node.
- If an agent doesn't stop within 60 seconds while `pmm-agent` restarts or removes it, `pmm-agent` stops waiting for that agent and applies the rest of the configuration.
- If `pmm-agent` can't process the requests from PMM Server for 2 minutes, it drops the connection and reconnects.

The recovery is part of PMM Client. On PMM Client 3.9 and earlier, restart `pmm-agent` to recover: run `sudo systemctl restart pmm-agent`, or restart the PMM Client container.

The following messages in the `pmm-agent` log show that a recovery took place:

| Message | Meaning |
|---------|---------|
| `Agent <AGENT_ID> did not report itself stopped, proceeding without it.` | The agent didn't stop within 60 seconds. `pmm-agent` continued without it. |
| `Request queue full for 2m0s, giving up on the connection.` | `pmm-agent` couldn't keep up with PMM Server for 2 minutes and reconnects. |

If the first message keeps appearing for the same agent ID, that agent doesn't stop. Before you restart `pmm-agent`, collect a goroutine dump on the PMM Client host and share it with Percona Support together with the `pmm-admin summary` archive:

```bash
curl -s "http://127.0.0.1:7777/debug/pprof/goroutine?debug=2" > pmm-agent-goroutines.txt
pmm-admin summary
```

If you changed the `pmm-agent` listen port, replace `7777` with that port.

## Connection difficulties

### Passwords

When adding a service, the host might not be detected if the password contains special symbols (e.g., `@`, `%`, etc.).

In such cases, you should convert any password, replacing special characters with their escape sequence equivalents.

One way to do this is to use the [`encodeURIComponent`](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/encodeURIComponent) JavaScript function in your browser's web console (commonly found under a *Development Tools* menu). Run the function with your password as the parameter. For example:

```js
> encodeURIComponent("s3cR#tpa$$worD")
```

will give:

```txt
"s3cR%23tpa%24%24worD"
```

### Password change

When adding clients to the PMM Server, you use the `admin` user. However, if you change the password for the admin user from the PMM UI, then the clients will not be able to access PMM due to authentication issues. Also, Grafana will lock out the admin user due to multiple unsuccessful login attempts.

In such a scenario, use [Service Accounts](../api/authentication.md#service-accounts-authentication) for authentication. You can use Service Accounts as a replacement for basic authentication and API keys.

### Server startup issues

#### PMM Server inaccessible after Docker restart on macOS Sequoia

After restarting Docker, PMM dashboard returns *500 Internal Server Error*.

##### Root cause

On macOS Sequoia 15.7.1 with Docker Desktop 4.49.0, PostgreSQL and Grafana may fail to start or not be able to be gracefully stopped, therefore transitioning to a FATAL state.

To verify, run the following command that checks service status. If both `postgresql` and `grafana` display their status as `BACKOFF` or `FATAL`, this confirms the issue:

```bash
docker exec pmm-server supervisorctl status
```

##### Solution

- upgrade Docker Desktop to a newer version or downgrade it to v4.48.0 (test with your specific setup)
- consider using PMM Server on Linux or cloud environments for production

## Log file locations

PMM Server stores all component logs in the `/srv/logs/` directory inside the container.

To see which log files are available, run:

=== "Docker"

    ```bash
    docker exec pmm-server ls /srv/logs/
    ```

=== "Podman"

    ```bash
    podman exec pmm-server ls /srv/logs/
    ```

To follow a log file in real time, use `tail -f`. For example, to monitor QAN API logs:

=== "Docker"

    ```bash
    docker exec pmm-server tail -f /srv/logs/qan-api2.log
    ```

=== "Podman"

    ```bash
    podman exec pmm-server tail -f /srv/logs/qan-api2.log
    ```