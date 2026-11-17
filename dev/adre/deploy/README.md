# PMM Assistant prototype — install

Try the PMM Assistant prototype (AI chat over your PMM data, Investigations and the Logs page) on a Linux test
machine with one script. It is a prototype build based on PMM 3.10, not a PMM release, so please use a test machine.

## Quick start

You need Docker or Podman (rootless works), `curl`, `openssl`, and port 443 free. The script uses Docker when
your user can talk to its daemon, else Podman, else Docker through `sudo`; set `DOCKER=podman` (or `docker`) to choose.

```bash
mkdir -p ~/pmm-assistant && cd ~/pmm-assistant
curl -fsSLO https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/install.sh
bash install.sh
```

After a few minutes it prints the PMM address. Log in as `admin`; the password is in `~/pmm-assistant/.env`
(`PMM_ADMIN_PASSWORD`). The browser warns about the self-signed certificate; that is expected.

Then add an AI model: **Configuration → Assistant → Deployment → Models & Keys → Add model** (a name, a LiteLLM model
id such as `anthropic/claude-sonnet-5-5`, and your API key) → **Save models**. Open **Assistant** from the left menu
and ask a question. Or add the model during install:

```bash
MODEL_NAME=claude-sonnet LITELLM_MODEL=anthropic/claude-sonnet-5-5 LLM_API_KEY=<your key> bash install.sh
```

**Rootless Podman** cannot publish port 443 until the host allows it, as in
[PMM's Podman guide](https://docs.percona.com/percona-monitoring-and-management/3/install-pmm/install-pmm-server/deployment-options/podman/index.html):
`echo net.ipv4.ip_unprivileged_port_start=443 | sudo tee /etc/sysctl.d/99-pmm.conf && sudo sysctl -p /etc/sysctl.d/99-pmm.conf`.
Without `sudo`, use another port: `PMM_HTTPS_PORT=8443 bash install.sh`. Rootless containers start again after a
reboot only with `loginctl enable-linger $USER` and `systemctl --user enable podman-restart.service`.

### Already running a PMM Server on that machine?

The script can add only the AI parts (HolmesGPT and the Grafana image renderer) next to it, with Docker or Podman:

```bash
mkdir -p ~/pmm-assistant/holmes-config && cd ~/pmm-assistant
curl -fsSLO https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/install.sh
```

Put the server's admin password in `~/pmm-assistant/.env` as a line `PMM_ADMIN_PASSWORD=<password>`, then run:

```bash
bash install.sh --existing-server        # or --existing-server=<name> if the container is not called pmm-server
```

The script never changes the admin password and checks the container before it changes anything. If the server
runs stock PMM, or lacks settings a container only takes when it is created, it stops and says how to recreate it
with the prototype image, the same data volume and ports, and what is missing:

- for a container you started with `docker run` or `podman run`, a ready `run` command;
- for a Podman container that a systemd unit runs (as in PMM's Podman guide), the lines to add to the unit's env
  file and its `podman run` line, then `systemctl --user daemon-reload && systemctl --user restart pmm-server`.

Do that, then run `bash install.sh --existing-server` again.

Switching a stock PMM's data to the prototype image is one-way: going back to stock PMM or upgrading to a later
official release is not supported. Use a test server, or back up its data volume first.

### Monitor databases and collect their logs

Install the PMM Client of the same build on each database host and register it with PMM as usual. `install.sh`
prints that client's Docker image and tarball at the end; its tag is the tag of the server image
(`perconalab/pmm-client-fb:PR-4259-<build>`).

To collect a database's logs, run this on that host, with the log file path and its parser preset:

```bash
pmm-admin add otel logs --log-sources=/var/log/mysql/error.log:mysql_error
```

Several files go in one comma-separated list, for example
`--log-sources=/var/log/mysql/error.log:mysql_error,/var/lib/pgsql/data/log/postgresql.log:postgres`. Presets for
databases: `mysql_error`, `postgres`, `mongodb`, `syslog_mysql_systemd`, or `raw` for no parsing (all presets are under
**Settings → OTEL**). The client must be able to read the file; with the Docker client, mount the log directory
into the container. Within a minute the lines show on the **Logs** page (left menu), where you can filter them and
group them into patterns. PMM Server's own logs are collected automatically.

### Update to a newer build

Download `install.sh` again and re-run it (it remembers `--existing-server`). A new install moves PMM Server to the
build the script names and keeps its data. With `--existing-server`, the script says when a newer build exists;
recreate the container from that image with the same options. A `PMM_IMAGE` line in `.env` pins the build; remove
it to follow updates. A build that moves to a new PMM release (as the 3.9 to 3.10 move did) needs a new server;
its announcement says so.

### Remove

Run `bash install.sh --remove` in `~/pmm-assistant`. It removes HolmesGPT, the renderer and, for a new install,
PMM Server; a server you started yourself keeps running. PMM's data volume stays: the script prints the command
that deletes it. Then delete the folder.

## How install.sh works

`install.sh`, in order: generates a self-signed TLS cert for HolmesGPT; starts `pmm-server` and
`grafana-image-renderer` on the `pmm` network and sets the admin password; picks the PMM URL HolmesGPT calls back
on (`https://<host IP>` when containers can reach it, else `https://pmm-server:8443`); enables the Assistant and
mints `PMM_API_TOKEN` + `HOLMES_API_KEY` through the PMM API; starts `holmesgpt` (HTTPS on 5050) and checks that PMM
can reach it. It runs the containers with `docker run` or `podman run`, no Compose. Each carries the labels
`pmm-assistant.dir` (this directory) and `pmm-assistant.config` (a hash of its settings): a re-run keeps a container
whose settings did not change and recreates the others; HolmesGPT is always recreated so it reads the tokens PMM
rendered. Re-running is safe: the cert, password and tokens are kept and models are upserted. With rootless Podman,
PMM Server runs with `--userns=keep-id:uid=1000,gid=1000` as in PMM's Podman guide, so the files PMM writes into the
shared directory belong to your user.

## Existing-server mode in detail

`install.sh --existing-server[=CONTAINER]` first checks the container and changes nothing until the check passes.
It never changes the admin password.

- **Stops with an error** when the container is missing or stopped, the password is wrong, or a `holmesgpt`
  container installed from another directory exists.
- **Asks for a recreate** (exit code 2) when the container lacks something only set when a container is created:
  an image with the Assistant (a stock PMM server is switched to `PMM_IMAGE`; that move is one-way, because the
  prototype adds its own database changes, so going back to stock PMM or upgrading to a later official release is
  not supported), the shared config dir (a host directory mounted at `ADRE_CONFIG_DIR`, default `/holmes-config`),
  `GF_RENDERING_SERVER_URL`, a `GF_RENDERING_CALLBACK_URL` that answers on the container's network address
  (Grafana listens on `127.0.0.1:3000` only, so use nginx's `http://<container>:8080/graph/`), a published HTTPS
  port, or a network other than `host`. A restart is not enough. For a container started by hand it prints a draft
  `run` command built from the container's image, ports and volumes, so the data volume is kept; the container's
  own `-e` options go to `./<container>.env` (mode 0600) for its `--env-file`, so no value is printed. For a Podman
  container a systemd unit starts (Podman's `PODMAN_SYSTEMD_UNIT` label), the unit recreates the container on every
  start, so it prints the env-file lines and `podman run` options to add to the unit instead.
- **Shares the server's network.** HolmesGPT and the renderer join the server's own network (`pmm_default` in PMM's
  Podman guide). A server on the default network only is connected to a new `pmm` network, unless a systemd unit
  runs it: then the unit needs `--network pmm`, and the check says so.
- **Warns** when in-app updates are on (`PMM_ENABLE_UPDATES=false` is missing).

When the check passes it starts the renderer (unless one is already running, or the server points to another
renderer), then provisions the Assistant and starts HolmesGPT as a new install does. A plain `bash install.sh`
refuses to start when a `pmm-server` container it did not create exists, and points to `--existing-server`. The
container name is saved in `./.env` as `PMM_EXISTING_SERVER`, so a plain re-run stays in this mode.

## Settings (all optional)

Defaults are in [`.env.example`](.env.example); override them in `./.env` or on the command line.

- **`PMM_IMAGE`** — a PMM Server image with the Assistant (a Feature Build of `tibi-holmes`); **the official
  `percona/pmm-server:3` does not have it** (`/v1/adre/*` → 404). The default is `DEFAULT_PMM_IMAGE` in
  `install.sh`, the only place that names the build: after a new Feature Build passes, change it there.
  `install.sh` does not save the images in `.env`, so setting `PMM_IMAGE` there pins an install to a build.
- **`HOLMES_IMAGE`** — `>= 0.36.0` (native HTTPS + `/api/admin/reload`); defaults to
  `docker.io/robustadev/holmes:0.45.0` ([release](https://github.com/HolmesGPT/holmesgpt/releases/tag/0.45.0)). Pin a
  digest for prod. Since 0.43.0 Holmes turns TodoWrite planning off by default; PMM opts Investigation mode back in
  via `behavior_controls`. **`RENDERER_IMAGE`** — the Grafana Image Renderer image.
- **`DOCKER`** — the container CLI (`docker`, `podman`, `sudo docker`); saved in `.env` so re-runs use the same one.
- **`PMM_HTTPS_PORT`** (new install, default `443`) and **`PMM_VOLUME`** (new install, default `pmm-data`).
- **`PMM_ADMIN_PASSWORD`**, **`PMM_CALLBACK_URL`**, **`HOLMES_CONFIG_DIR`**, **`PMM_LOCAL`** — see `.env.example`.
- **`MODEL_NAME` / `LITELLM_MODEL` / `LLM_API_KEY`** — all three or none.
- **`FAST_MODEL_NAME` / `FAST_LITELLM_MODEL`** (+ optional `FAST_LLM_API_KEY`) — a cheaper model for Fast chat,
  the default mode; `MODEL_NAME` then runs Investigation mode only. Every request carries about 23k tokens
  of tool definitions and instructions, so the per-token price of the Fast model sets the cost of a simple
  question.

## What is shared

One host directory (`$HOLMES_CONFIG_DIR`, default `./holmes-config`) is bind-mounted into both containers (with
`:z`, so SELinux hosts let both read it) and is the ONLY thing they share:

```
holmes-config/
  .env             # PMM_URL, PMM_API_TOKEN (minted), HOLMES_API_KEY  (rendered by PMM, 0600)
  config.yaml      # toolsets
  model_list.yaml  # models with provider api_key
  skills/<name>/SKILL.md
  certs/tls.crt    # self-signed cert holmesgpt serves HTTPS with (install.sh)
  certs/tls.key
```

`install.sh` makes this dir `0777` because PMM renders into it as uid 1000 (`pmm`); the secret files
themselves stay `0600`. PMM's `/srv` data volume is never shared with Holmes.

## Applying changes

Edit in **PMM UI → Configuration → Assistant → Deployment** and **Save**. PMM renders to the shared dir and hot-reloads
Holmes via `POST /api/admin/reload` (`ENABLE_ADMIN_API=true` is set), so **config.yaml / model_list.yaml / skills
changes apply with no restart**. Only bootstrap `.env` changes (PMM URL / secrets — the Provisioning tab) need
HolmesGPT recreated: re-run `bash install.sh`.

## Notes / caveats

- **HTTPS is native** (`HOLMES_SSL_CERTFILE`/`KEYFILE`; verified: "Holmes API serving HTTPS on 0.0.0.0:5050"). PMM
  connects with **Skip TLS verification** (install.sh sets it, since the cert is self-signed). For a CA-signed cert,
  drop skip-verify. mTLS (`HOLMES_SSL_CA_CERTS`) is **not usable yet** — PMM's client doesn't present a client cert.
- **No `.py` patches, no init containers.** The old `database.py` / `openai_formatting.py` mounts and the
  TLS patch script were pre-0.36 workarounds: Holmes now serves TLS natively and emits valid strict tool
  schemas itself (leave `TOOL_SCHEMA_NO_PARAM_OBJECT_IF_NO_PARAMS` unset), and the ClickHouse-HTTP one is an
  opt-in config flag. Do not mount old copies of those files into a newer image.
- **First PMM start can take a few minutes** (fresh `/srv` init); `install.sh` waits up to ~10 minutes.
- **Updates are disabled** (`PMM_ENABLE_UPDATES=false`): the in-app updater would replace this build with a
  stock PMM release, which has no Assistant. Move to a newer build as described in
  [Update to a newer build](#update-to-a-newer-build).
- **Older installs made with Compose** (`compose.yaml` next to `install.sh`) are picked up: the script treats their
  containers as its own and keeps the data volume, recreating each container once with its own settings.
- **k8s toolset:** to use it, mount your kubeconfig into the `holmesgpt` container at `/root/.kube/config`
  (add the `-v` to `holmes_args` in `install.sh`).
- **Single-host lab bundle — not hardened for shared hosts.** The admin password is generated and stored in `./.env` (0600), and note that `install.sh` makes the shared config dir `0777` and that secrets (the minted PMM/Holmes tokens, the self-signed TLS key, and `model_list.yaml` with your LLM key) live in it. That's fine on a single-admin host; on a multi-user host, tighten those perms. `install.sh` sends request bodies with secrets to the PMM API on stdin; the admin password itself is passed to `curl` as `-u`, which other users of the host can see in `ps` while it runs.

## Slack auto-investigate

The Assistant chat works after install. To also get **auto-investigation of firing alerts posted into the
alert's Slack thread**, configure the Slack bot + Grafana contact point — see
[../../../documentation/docs/use/ai-features/adre-slack-bot.md](../../../documentation/docs/use/ai-features/adre-slack-bot.md).
The key point: auto-investigate is triggered by the Slack **alert scrape**, which only acts on messages that
**@-mention the PMM bot** — so the Grafana Slack contact point's **"Mention Users"** must be set to the PMM bot's
user id. There is no webhook and no reconciliation poll; the mention scrape is the sole trigger.
