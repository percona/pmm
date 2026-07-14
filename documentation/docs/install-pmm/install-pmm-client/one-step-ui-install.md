# One-step PMM Client install from UI

Use the **Install PMM Client** wizard to generate a single command that installs `pmm-client`, registers the node with PMM Server, and adds one monitored service.

## Before you start

- PMM Server must be reachable from the target node (default port `443`; whatever you set in **PMM host** is used in `PMM_SERVER_URL`).
- The node user running the command needs `sudo` access (or run it as `root`, e.g. inside a container).
- A short-lived service token is minted from the UI on demand — you do not need to provision one beforehand. The Grafana **Install PMM Client** service account is **Admin** org role and **expires 15 minutes after generation**; treat the URL like a password.

## Generate the command

1. In PMM UI, open **Inventory → Install PMM Client**.
2. Choose **Technology**: `MySQL`, `PostgreSQL`, `MongoDB`, or `Valkey`.
3. Click **Generate short-lived token**. The countdown chip shows the remaining lifetime.
4. Copy the **Generated command** and run it on your database server with `sudo` before the token expires. By default the command uses **prompt on node** mode: the script downloads to `/tmp/install-pmm-client.sh`, then asks for the DB user and password on the server (they are not embedded in the command).
5. Optional — open **Advanced options** for node name/address, DB host/port, service name, MySQL query source (QAN), PostgreSQL database, MongoDB auth DB, or **Use insecure TLS**.
6. Optional — enable **Running in CI/automation?** if you need credentials embedded in the command instead of prompted on the node:
   - **Include env variables** (recommended for `curl | bash`): credentials in the environment of the spawned shell.
   - **Pass as script flags**: credentials as `--db-user` / `--db-password` script arguments.
   - **Prompt on node** remains available here for automation that still allocates a TTY.

**Advanced options** also contains **Force re-register node** — use only to recover from a broken first install (it removes the existing node and all its services on PMM Server). Do not enable it when adding another database instance on the same host.

Example (env mode, MySQL with Performance Schema QAN):

```bash
curl -fsSLk 'https://<pmm-host>/pmm-static/install-pmm-client.sh' | sudo -E env \
  PMM_SERVER_URL='https://service_token:<TOKEN>@<pmm-host>' \
  TECH='mysql' \
  DB_USER='pmm' \
  DB_PASSWORD='secret' \
  DB_QUERY_SOURCE='perfschema' \
  bash -s -- \
  --pmm-server-insecure-tls
```

Example (env mode, matches what the wizard renders):

```bash
curl -fsSLk 'https://<pmm-host>/pmm-static/install-pmm-client.sh' | sudo -E env \
  PMM_SERVER_URL='https://service_token:<TOKEN>@<pmm-host>' \
  TECH='mysql' \
  DB_USER='pmm' \
  DB_PASSWORD='secret' \
  bash -s -- \
  --pmm-server-insecure-tls
```

Example (prompt mode — credentials never appear in the rendered command):

```bash
curl -fsSLk -o '/tmp/install-pmm-client.sh' 'https://<pmm-host>/pmm-static/install-pmm-client.sh'
sudo -E bash '/tmp/install-pmm-client.sh' \
  --pmm-server-url 'https://service_token:<TOKEN>@<pmm-host>' \
  --tech 'mysql' \
  --pmm-server-insecure-tls
```

The second line runs `bash` against a file (not against a pipe), so `sudo` keeps stdin connected to your terminal. The script then asks twice — once for the DB user, once for the DB password (silent input) — before running `pmm-admin add`. Fields you set under **Advanced options** (host, port, service name, MySQL query source, MongoDB auth DB, PostgreSQL database) are passed as `--db-*` flags so you only have to type two things on the node.

Notes on the rendered command:

- `curl -fsSLk` (with `-k`) is emitted only when **Use insecure TLS** is on; with a properly signed PMM Server certificate the wizard drops the `-k`.
- TLS-skip on the PMM Server side is controlled by the `--pmm-server-insecure-tls` script flag (passed after `bash -s --`, or as an argument to `bash <path>` in prompt mode). The script also accepts `PMM_SERVER_INSECURE_TLS=1` as an env var if you build the command by hand.
- `sudo -E env VAR=... bash -s --` is the standard shape for env/flags modes; `-E` preserves your shell's exports while the explicit `VAR=...` list gets handed to `bash`'s environment (and therefore to the script).
- Prompt mode uses `sudo -E bash <path> ...` instead of `sudo -E env … bash -s --`: there is no inline env block in the copied command, but `-E` still forwards your shell exports (e.g. `DB_USER` / `DB_PASSWORD`) so credentials can be supplied without prompts or appearing in the command string. Stdin stays on your TTY the same way as plain `sudo bash`.

## What the script does

The script available at `/pmm-static/install-pmm-client.sh` performs:

1. Installs `pmm-client` using the OS package manager (RHEL-compatible or Debian-compatible hosts).
2. Ensures `pmm-agent` is running (starts it via `systemd` when available, otherwise `nohup` in the background).
3. Runs `pmm-admin config` against your PMM server to register the node and persist the agent identity — **skipped automatically** when `pmm-agent` is already set up on the node (use **Force re-register node** / `--force` only when you need to replace the existing registration).
4. Runs `pmm-admin add <technology>` using your selected options.

When **Service name** is left empty in the wizard, the script picks `<hostname>-<tech>` (for example `db1-mysql`). If you set **DB port** (or the effective port is not the technology default), the script appends `-<port>` (for example `db1-mysql-3307`) so multiple instances on one node get distinct names.

For **MySQL**, set **Query source (QAN)** to `slowlog`, `perfschema`, or `none`. The script passes it to `pmm-admin add mysql` as `--query-source`. When omitted, `pmm-admin` uses its default (`slowlog`). See [Connect MySQL databases to PMM](connect-database/mysql/mysql.md#quick-setup) for permission requirements per source.

## Multiple database instances on one node

Use the wizard **once per instance** on the same host:

1. **First instance** — generate a token and run the full command (install, register node, add service). Include **PMM host**, token, and credentials as usual.
2. **Additional instances** — generate a new token, set a different **DB port** (and credentials if they differ), and run the command again on the same node. You do **not** need **Force re-register node**; the script detects an already-configured `pmm-agent` and skips `pmm-admin config`, then runs only `pmm-admin add`.
3. Leave **Service name** empty unless you want a custom name — the script suffixes the port when needed to avoid name clashes.

Example — second MySQL on port `3307` (prompt mode; DB credentials are prompted on the node):

```bash
curl -fsSLk -o '/tmp/install-pmm-client.sh' 'https://<pmm-host>/pmm-static/install-pmm-client.sh'
sudo -E bash '/tmp/install-pmm-client.sh' \
  --pmm-server-url 'https://service_token:<TOKEN>@<pmm-host>' \
  --tech 'mysql' \
  --db-port '3307' \
  --pmm-server-insecure-tls
```

After the first run, `pmm-admin config` is skipped even though the command still includes `--pmm-server-url` (required for the first run and harmless on later runs). For additional instances you may omit `--pmm-server-url` if you build the command by hand.

**Warning:** **Force re-register node** (`--force`) removes the existing node **and all services** on PMM Server before registering again. Use it only to recover from a broken first registration — not when adding another database instance.

## Security notes

- Generated tokens are tied to Grafana service accounts minted as **Admin** org role and live for **15 minutes** — generate, run, done. There is no way to extend the lifetime from the UI.
- The default **prompt on node** flow keeps DB credentials off the clipboard. Enable **Running in CI/automation?** only when you need env/flags mode; credentials may then appear in the command line or process environment briefly on the target node.
- Avoid copy-pasting the command into chat/issue trackers; the embedded service token is a credential. (In prompt mode the DB credentials are not in the command, but the PMM service token still is.)

## Troubleshooting

- **curl / browser returns `404`** on URLs like `/graph/…` — PMM Web UI lives under **`/pmm-ui/`**. Use paths such as `/pmm-ui/graph/inventory/nodes`, not `/graph/inventory/nodes`. This matches what the browser loads (see address bar vs. truncated copy-pastes).

- **TLS handshake errors against PMM Server** — turn on **Use insecure TLS** in the wizard (sets the `--pmm-server-insecure-tls` script flag). The wizard also adds `-k` to `curl` so the script download itself succeeds.
- **Package install fails** — verify outbound access to the Percona repositories (`repo.percona.com`).
- **`pmm-admin add` fails (auth, name conflict, etc.)** — on the **first** run, the node was already registered by `pmm-admin config`. Re-run with **Force re-register node** enabled (this passes `--force` to `pmm-admin config`, which removes the previous node and its services on the server before re-registering). You may also have to fix the database credentials before retrying. On **additional instances** on an already-registered node, a name conflict usually means the service name collides — set a unique **Service name** or a distinct **DB port** so the script default includes a port suffix.
- **`pmm-agent is not running`** — happens in containers without `systemd`. The script auto-starts it via `nohup` and writes logs to `/var/log/pmm-agent.log`; check there.
- **`hostname: command not found`** — only on extremely minimal images; the script falls back to `$HOSTNAME`/`uname -n`/`/etc/hostname` and finally `node`.
- **Prompt mode does not actually prompt** — the script's noninteractive guard fires when stdin is not a TTY, e.g. when the prompt-mode command is invoked through `ssh host '<paste>'` (no allocated TTY) or through automation. Run it from an interactive shell on the node, or enable **Running in CI/automation?** and use **Include env variables** / **Pass as script flags** instead.
- **Cleanup after prompt mode** — the downloaded script lives at `/tmp/install-pmm-client.sh` after a successful install. It is harmless (no embedded secrets), but if you want it gone: `rm -f /tmp/install-pmm-client.sh`.
