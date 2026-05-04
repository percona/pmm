# One-step PMM Client install from UI

Use the **Install PMM Client** wizard to generate a single command that installs `pmm-client`, registers the node with PMM Server, and adds one monitored service.

## Before you start

- PMM Server must be reachable from the target node (default port `443`; whatever you set in **PMM host** is used in `PMM_SERVER_URL`).
- The node user running the command needs `sudo` access (or run it as `root`, e.g. inside a container).
- A short-lived service token is minted from the UI on demand — you do not need to provision one beforehand. The Grafana **Install PMM Client** service account is **Admin** org role and **expires 15 minutes after generation**; treat the URL like a password.

## Generate the command

1. In PMM UI, open **Inventory → Install PMM Client**.
2. Choose technology: `MySQL`, `PostgreSQL`, `MongoDB`, or `Valkey`.
3. Click **Generate short-lived token**. The countdown chip shows the remaining lifetime.
4. Select the credentials mode:
   - **Prompt on node (downloads script first, asks for DB user/password)** (**default**): the wizard renders a **two-step** command — `curl … -o /tmp/install-pmm-client.sh '<url>'` followed by `sudo -E bash /tmp/install-pmm-client.sh …`. Reading the script from disk (instead of piping it from `curl`) keeps stdin attached to your terminal, so the script can prompt you for the DB user and password. **`sudo -E`** preserves your environment into the root shell: if `DB_USER` / `DB_PASSWORD` (or per-tech `MYSQL_*`, `POSTGRESQL_*`, …) are already exported, the script uses them and **does not prompt**. Use this when you do not want credentials in the copied command line or process list from flags alone.
   - **Include env variables (recommended for `curl | bash`)**: credentials are passed in the environment of the spawned shell. Use this when you want the classic one-liner pipeline.
   - **Pass as script flags**: credentials are passed as `--db-*` script arguments instead of env vars. Same security profile as env mode, just a different surface.
5. Fill in the optional fields you need (node name/address, DB host/port, service name, MongoDB auth DB, PostgreSQL database).
6. Copy the generated command and run it on the target node before the token expires.

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

The second line runs `bash` against a file (not against a pipe), so `sudo` keeps stdin connected to your terminal. The script then asks twice — once for the DB user, once for the DB password (silent input) — before running `pmm-admin add`. Optional fields you fill in the wizard (host, port, service name, MongoDB auth DB, PostgreSQL database) are still passed as `--db-*` flags so you only have to type two things.

Notes on the rendered command:

- `curl -fsSLk` (with `-k`) is emitted only when **Use insecure TLS** is on; with a properly signed PMM Server certificate the wizard drops the `-k`.
- TLS-skip on the PMM Server side is controlled by the `--pmm-server-insecure-tls` script flag (passed after `bash -s --`, or as an argument to `bash <path>` in prompt mode). The script also accepts `PMM_SERVER_INSECURE_TLS=1` as an env var if you build the command by hand.
- `sudo -E env VAR=... bash -s --` is the standard shape for env/flags modes; `-E` preserves your shell's exports while the explicit `VAR=...` list gets handed to `bash`'s environment (and therefore to the script).
- Prompt mode uses `sudo -E bash <path> ...` instead of `sudo -E env … bash -s --`: there is no inline env block in the copied command, but `-E` still forwards your shell exports (e.g. `DB_USER` / `DB_PASSWORD`) so credentials can be supplied without prompts or appearing in the command string. Stdin stays on your TTY the same way as plain `sudo bash`.

## What the script does

The script available at `/pmm-static/install-pmm-client.sh` performs:

1. Installs `pmm-client` using the OS package manager (RHEL-compatible or Debian-compatible hosts).
2. Ensures `pmm-agent` is running (starts it via `systemd` when available, otherwise `nohup` in the background).
3. Runs `pmm-admin config` against your PMM server to register the node and persist the agent identity.
4. Runs `pmm-admin add <technology>` using your selected options.

## Security notes

- Generated tokens are tied to Grafana service accounts minted as **Admin** org role and live for **15 minutes** — generate, run, done. There is no way to extend the lifetime from the UI.
- Env mode and flags mode put credentials into the shell command line and the spawned process environment. On a shared node, that may be visible in `ps`/`/proc` to other users for a moment. **Prompt mode** avoids this entirely: the rendered command never contains the DB user or password, and the script reads them straight from your terminal once it is running on the node.
- Avoid copy-pasting the command into chat/issue trackers; the embedded service token is a credential. (In prompt mode the DB credentials are not in the command, but the PMM service token still is.)

## Troubleshooting

- **curl / browser returns `404`** on URLs like `/graph/…` — PMM Web UI lives under **`/pmm-ui/`**. Use paths such as `/pmm-ui/graph/inventory/nodes`, not `/graph/inventory/nodes`. This matches what the browser loads (see address bar vs. truncated copy-pastes).

- **TLS handshake errors against PMM Server** — turn on **Use insecure TLS** in the wizard (sets the `--pmm-server-insecure-tls` script flag). The wizard also adds `-k` to `curl` so the script download itself succeeds.
- **Package install fails** — verify outbound access to the Percona repositories (`repo.percona.com`).
- **`pmm-admin add` fails (auth, name conflict, etc.)** — the node was already registered by `pmm-admin config`. Re-run with **Force re-register node** enabled (this passes `--force` to `pmm-admin config`, which removes the previous node and its services on the server before re-registering). You may also have to fix the database credentials before retrying.
- **`pmm-agent is not running`** — happens in containers without `systemd`. The script auto-starts it via `nohup` and writes logs to `/var/log/pmm-agent.log`; check there.
- **`hostname: command not found`** — only on extremely minimal images; the script falls back to `$HOSTNAME`/`uname -n`/`/etc/hostname` and finally `node`.
- **Prompt mode does not actually prompt** — the script's noninteractive guard fires when stdin is not a TTY, e.g. when the prompt-mode command is invoked through `ssh host '<paste>'` (no allocated TTY) or through automation. Run it from an interactive shell on the node, or use **Include env variables** / **Pass as script flags** mode instead.
- **Cleanup after prompt mode** — the downloaded script lives at `/tmp/install-pmm-client.sh` after a successful install. It is harmless (no embedded secrets), but if you want it gone: `rm -f /tmp/install-pmm-client.sh`.
