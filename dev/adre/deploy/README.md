# PMM Assistant prototype — install

Try the PMM Assistant prototype (AI chat over your PMM data, Investigations and the Logs page) on a Linux test
machine with one script. It is a prototype build based on PMM 3.10, not a PMM release, so please use a test machine.

## Quick start

You need Docker with the Compose plugin, `curl`, `openssl`, and port 443 free. If your user needs `sudo` for
Docker, the script detects it.

```bash
mkdir -p ~/pmm-assistant && cd ~/pmm-assistant
curl -fsSL --remote-name-all https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/{compose.yaml,install.sh}
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

### Already running a PMM Server on that machine?

The script can add only the AI parts (HolmesGPT and the Grafana image renderer) next to it:

```bash
mkdir -p ~/pmm-assistant/holmes-config && cd ~/pmm-assistant
curl -fsSL --remote-name-all https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/{compose.yaml,install.sh}
```

Put the server's admin password in `~/pmm-assistant/.env` as a line `PMM_ADMIN_PASSWORD=<password>`, then run:

```bash
bash install.sh --existing-server        # or --existing-server=<name> if the container is not called pmm-server
```

The script never changes the admin password and checks the container before it changes anything. If the server
runs stock PMM, or lacks settings Docker only applies when a container is created, it stops and prints a
`docker run` command that recreates it with the prototype image, the same data volume and ports, and what is
missing. Run that command, then run `bash install.sh --existing-server` again.

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

Download the two files again and re-run `bash install.sh` (it remembers `--existing-server`). A new install
moves PMM Server to the build `compose.yaml` names and keeps its data. With `--existing-server`, the script says
when a newer build exists; recreate the container from that image with the same options. A `PMM_IMAGE` line in
`.env` pins the build; remove it to follow updates. A build that moves to a new PMM release (as the 3.9 to 3.10
move did) needs a new server; its announcement says so.

### Remove

New install: run `docker compose down -v` in `~/pmm-assistant`, then delete that folder. This deletes PMM's data.

With `--existing-server`: run `docker network disconnect pmm pmm-server`, then `docker compose down` in
`~/pmm-assistant`. HolmesGPT and the renderer are removed; your PMM Server and its data stay.

## How install.sh works

`install.sh`, in order: generates a self-signed TLS cert for HolmesGPT; starts `pmm-server` and
`grafana-image-renderer` and sets the admin password; picks the PMM URL HolmesGPT calls back on
(`https://<host IP>` when containers can reach it, else `https://pmm-server:8443`); enables the Assistant and mints
`PMM_API_TOKEN` + `HOLMES_API_KEY` through the PMM API; starts `holmesgpt` (HTTPS on 5050) and checks that
PMM can reach it. Re-running it is safe: the cert, password and tokens are kept and models are upserted.

> Start Holmes through `install.sh` (or recreate it after provisioning with
> `docker compose up -d --force-recreate holmesgpt`): a plain `docker compose up -d` before provisioning
> starts Holmes without the tokens.

## Existing-server mode in detail

`install.sh --existing-server[=CONTAINER]` first checks the container and changes nothing until the check passes.
It never changes the admin password.

- **Stops with an error** when the container is missing or stopped, the password is wrong, or a `holmesgpt`
  container installed from another directory exists.
- **Asks for a recreate** (exit code 2) when the container lacks something Docker only sets when a container is
  created: an image with the Assistant (a stock PMM server gets a command that switches it to `PMM_IMAGE`; that
  move is one-way, because the prototype adds its own database changes, so going back to stock PMM or upgrading to
  a later official release is not supported), the shared config dir (a host directory mounted at `ADRE_CONFIG_DIR`, default `/holmes-config`),
  `GF_RENDERING_SERVER_URL`, a `GF_RENDERING_CALLBACK_URL` that answers on the container's network address
  (Grafana listens on `127.0.0.1:3000` only, so use nginx's `http://<container>:8080/graph/`), a published HTTPS
  port, or a network other than `host`. A restart is not enough. It prints the `docker run` options to add and a
  draft command built from the container's image, ports and volumes, so the data volume is kept. The container's
  own `-e` options go to `./<container>.env` (mode 0600) for the draft's `--env-file`, so no value is printed.
- **Fixes on the fly** what needs no recreate: it connects the container to the `pmm` network.
- **Warns** when in-app updates are on (`PMM_ENABLE_UPDATES=false` is missing).

When the check passes it starts the renderer (unless one is already running, or the server points to another
renderer), then provisions the Assistant and starts HolmesGPT as a new install does. A plain `bash install.sh`
refuses to start when a `pmm-server` container it did not create exists, and points to `--existing-server`. The container name is saved
in `./.env` as `PMM_EXISTING_SERVER`, so a plain `bash install.sh` re-run stays in this mode. Run `docker compose`
commands for `holmesgpt` and `grafana-image-renderer` only: `docker compose up -d` without a service name would try
to create a second `pmm-server`.

## Settings (all optional)

Defaults are in [`.env.example`](.env.example); override them in `./.env` or on the command line.

- **`PMM_IMAGE`** — a PMM Server image with the Assistant (a Feature Build of `tibi-holmes`); **the official
  `percona/pmm-server:3` does not have it** (`/v1/adre/*` → 404). The default is the image in `compose.yaml`, the
  only place that names the build: after a new Feature Build passes, change it there and both files follow.
  `install.sh` does not save the images in `.env`, so setting `PMM_IMAGE` there pins an install to a build.
- **`HOLMES_IMAGE`** — `>= 0.36.0` (native HTTPS + `/api/admin/reload`); defaults to
  `robustadev/holmes:0.45.0` ([release](https://github.com/HolmesGPT/holmesgpt/releases/tag/0.45.0)). Pin a digest
  for prod. Since 0.43.0 Holmes turns TodoWrite planning off by default; PMM opts Investigation mode back in via
  `behavior_controls`.
- **`PMM_ADMIN_PASSWORD`**, **`PMM_CALLBACK_URL`**, **`HOLMES_CONFIG_DIR`** — see `.env.example`.
- **`MODEL_NAME` / `LITELLM_MODEL` / `LLM_API_KEY`** — all three or none.
- **`FAST_MODEL_NAME` / `FAST_LITELLM_MODEL`** (+ optional `FAST_LLM_API_KEY`) — a cheaper model for Fast chat,
  the default mode; `MODEL_NAME` then runs Investigation mode only. Every request carries about 23k tokens
  of tool definitions and instructions, so the per-token price of the Fast model sets the cost of a simple
  question.

## What is shared

One host directory (`$HOLMES_CONFIG_DIR`, default `./holmes-config`) is bind-mounted into both containers
and is the ONLY thing they share:

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

Edit in **PMM UI → Configuration → Assistant → Deployment** and **Save**. PMM renders to the shared dir and hot-reloads Holmes via
`POST /api/admin/reload` (`ENABLE_ADMIN_API=true` is set), so **config.yaml / model_list.yaml / skills changes
apply with no restart**. Only bootstrap `.env` changes (PMM URL / secrets — the Provisioning tab) need a
recreate:

```bash
docker compose up -d --force-recreate holmesgpt
```

## Notes / caveats

- **HTTPS is native** (`HOLMES_SSL_CERTFILE`/`KEYFILE` in `compose.yaml`; verified: "Holmes API serving HTTPS
  on 0.0.0.0:5050"). PMM connects with **Skip TLS verification** (install.sh sets it, since the cert is
  self-signed). For a CA-signed cert, drop skip-verify. mTLS (`HOLMES_SSL_CA_CERTS`) is **not usable yet** —
  PMM's client doesn't present a client cert.
- **No `.py` patches, no init containers.** The old `database.py` / `openai_formatting.py` mounts and the
  TLS patch script were pre-0.36 workarounds: Holmes now serves TLS natively and emits valid strict tool
  schemas itself (leave `TOOL_SCHEMA_NO_PARAM_OBJECT_IF_NO_PARAMS` unset), and the ClickHouse-HTTP one is an
  opt-in config flag. Do not mount old copies of those files into a newer image.
- **First PMM start can take a few minutes** (fresh `/srv` init); `install.sh` waits up to ~10 minutes.
- **Updates are disabled** (`PMM_ENABLE_UPDATES=false`): the in-app updater would replace this build with a
  stock PMM release, which has no Assistant. Move to a newer build as described in
  [Update to a newer build](#update-to-a-newer-build).
- **k8s toolset:** if you need it, add `- /path/to/kube_config:/root/.kube/config` to the `holmesgpt` volumes
  (omitted by default).
- **Single-host lab bundle — not hardened for shared hosts.** The admin password is generated and stored in `./.env` (0600), and note that `install.sh` makes the shared config dir `0777` and that secrets (the minted PMM/Holmes tokens, the self-signed TLS key, and `model_list.yaml` with your LLM key) live in it. That's fine on a single-admin host; on a multi-user host, tighten those perms. `install.sh` sends secrets to the PMM API on stdin, never as command-line arguments.

## Slack auto-investigate

The Assistant chat works after install. To also get **auto-investigation of firing alerts posted into the
alert's Slack thread**, configure the Slack bot + Grafana contact point — see
[../../../documentation/docs/use/ai-features/adre-slack-bot.md](../../../documentation/docs/use/ai-features/adre-slack-bot.md).
The key point: auto-investigate is triggered by the Slack **alert scrape**, which only acts on messages that
**@-mention the PMM bot** — so the Grafana Slack contact point's **"Mention Users"** must be set to the PMM bot's
user id. There is no webhook and no reconciliation poll; the mention scrape is the sole trigger.
