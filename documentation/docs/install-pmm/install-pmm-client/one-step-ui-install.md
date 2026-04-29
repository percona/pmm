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
   - **Include env variables (recommended for `curl | bash`)**: credentials are passed in the environment of the spawned shell. This is the default and works with the one-liner.
   - **Pass as script flags**: credentials are passed as `--db-*` script arguments instead of env vars. Same security profile as env mode, just a different surface.
   - **Prompt on node (TTY only)**: the script asks for credentials interactively. This **only works** if you save the script and run it from a real shell (`sudo bash ./install-pmm-client.sh ...`); piping from `curl` consumes stdin and the prompt cannot read your keyboard.
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

Notes on the rendered command:

- `curl -fsSLk` (with `-k`) is emitted only when **Use insecure TLS** is on; with a properly signed PMM Server certificate the wizard drops the `-k`.
- TLS-skip on the PMM Server side is controlled by the `--pmm-server-insecure-tls` script flag (passed after `bash -s --`). The script also accepts `PMM_SERVER_INSECURE_TLS=1` as an env var if you build the command by hand.
- `sudo -E env VAR=... bash -s --` is the standard shape; `-E` preserves your shell's exports while the explicit `VAR=...` list gets handed to `bash`'s environment (and therefore to the script).

## What the script does

The script available at `/pmm-static/install-pmm-client.sh` performs:

1. Installs `pmm-client` using the OS package manager (RHEL-compatible or Debian-compatible hosts).
2. Ensures `pmm-agent` is running (starts it via `systemd` when available, otherwise `nohup` in the background).
3. Runs `pmm-admin config` against your PMM server to register the node and persist the agent identity.
4. Runs `pmm-admin add <technology>` using your selected options.

## Security notes

- Generated tokens are tied to Grafana service accounts minted as **Admin** org role and live for **15 minutes** — generate, run, done. There is no way to extend the lifetime from the UI.
- Env mode and flags mode put credentials into the shell command line and the spawned process environment. On a shared node, that may be visible in `ps`/`/proc` to other users for a moment. Prompt mode avoids this but, as noted above, can only be used from a real terminal — not through `curl | bash`.
- Avoid copy-pasting the command into chat/issue trackers; the embedded service token is a credential.

## Troubleshooting

- **curl / browser returns `404`** on URLs like `/graph/…` — PMM Web UI lives under **`/pmm-ui/`**. Use paths such as `/pmm-ui/graph/inventory/nodes`, not `/graph/inventory/nodes`. This matches what the browser loads (see address bar vs. truncated copy-pastes).

- **TLS handshake errors against PMM Server** — turn on **Use insecure TLS** in the wizard (sets the `--pmm-server-insecure-tls` script flag). The wizard also adds `-k` to `curl` so the script download itself succeeds.
- **Package install fails** — verify outbound access to the Percona repositories (`repo.percona.com`).
- **`pmm-admin add` fails (auth, name conflict, etc.)** — the node was already registered by `pmm-admin config`. Re-run with **Force re-register node** enabled (this passes `--force` to `pmm-admin config`, which removes the previous node and its services on the server before re-registering). You may also have to fix the database credentials before retrying.
- **`pmm-agent is not running`** — happens in containers without `systemd`. The script auto-starts it via `nohup` and writes logs to `/var/log/pmm-agent.log`; check there.
- **`hostname: command not found`** — only on extremely minimal images; the script falls back to `$HOSTNAME`/`uname -n`/`/etc/hostname` and finally `node`.
