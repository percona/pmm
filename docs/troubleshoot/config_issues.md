
# Configuration issues

This section focuses on configuration issues, such as PMM-agent connection, adding and removing services for monitoring, and so on.

## Client-server connections

There are many causes of broken network connectivity.

The container is constrained by the host-level routing and firewall rules when using [using Docker](../install-pmm/install-pmm-server/index.md). For example, your hosting provider might have default `iptables` rules on their hosts that block communication between PMM Server and PMM Client, resulting in *DOWN* targets in VictoriaMetrics. If this happens, check the firewall and routing settings on the Docker host.

PMM can also generate diagnostics data that can be examined and/or shared with our support team to help solve an issue. You can get collected logs from PMM Client using the pmm-admin summary command.

Logs obtained in this way include PMM Client logs and logs received from the PMM Server, and stored separately in the `client` and `server` folders. The `server` folder also contains its `client` subfolder with the self-monitoring client information collected on the PMM Server.

Beginning with [PMM 2.4.0](../release-notes/2.4.0.md), there is a flag that enables the fetching of [`pprof`](https://github.com/google/pprof) debug profiles and adds them to the diagnostics data. To enable, run `pmm-admin summary --pprof`.

You can get PMM Server logs with either of these methods:

**Direct download**

In a browser, visit `https://<address-of-your-pmm-server>/logs.zip`.

**From Help menu**

To obtain the logs from the **Help** menu:
{.power-number}


1. Select <i class="uil uil-question-circle"></i> **Help** → <i class="uil uil-download-alt"></i> **PMM Logs**.

2. Click **PMM Logs** to retrieve PMM diagnostics data which can be examined and shared with our support team should you need help.

## Connection difficulties

**Passwords**

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

**Password change**

When adding clients to the PMM server, you use the `admin` user. However, if you change the password for the admin user from the PMM UI, then the clients will not be able to access PMM due to authentication issues. Also, Grafana will lock out the admin user due to multiple unsuccessful login attempts.

In such a scenario, use [API key](../api/authentication.md#api-keys-and-authentication) for authentication. You can use API keys as a replacement for basic authentication.



