# One-step PMM Client install from UI

Use the **Install PMM Client** wizard to generate a single command that installs `pmm-client`, registers the node, and adds one monitored service.

## Before you start

- PMM Server must be reachable from the target node on `443`.
- The node user running the command needs `sudo` access.
- You need a PMM token that works with `pmm-admin config --server-url`, for example:
  - `https://service_token:GLSA_...@<pmm-server>:443`

## Generate the command

1. In PMM UI, open **Inventory** -> **Install PMM Client**.
2. Choose technology: `MySQL`, `PostgreSQL`, `MongoDB`, or `Valkey`.
3. Select credentials mode:
   - **Prompt on node (recommended)**: password is typed interactively on the node.
   - **Include env variables**: password is passed through environment variables.
   - **Pass as script flags**: password is passed as script arguments.
4. Paste your service token and adjust optional fields (node name/address, DB host/port, service name).
5. Copy the generated command and run it on the target node.

Example (env mode):

```bash
curl -fsSL "https://<pmm-host>/pmm-static/install-pmm-client.sh" | sudo env \
  PMM_SERVER_URL='https://service_token:<TOKEN>@<pmm-host>:443' \
  PMM_SERVER_INSECURE_TLS=1 \
  TECH=mysql \
  DB_USER='pmm' \
  DB_PASSWORD='secret' \
  bash
```

## What the script does

The script available at `/pmm-static/install-pmm-client.sh` performs:

1. Install `pmm-client` using the OS package manager (RHEL-compatible or Debian-compatible hosts).
2. Run `pmm-admin config` against your PMM server.
3. Run `pmm-admin register`.
4. Run `pmm-admin add <technology>` using your selected options.

## Security notes

- Prefer **Prompt on node** for production to avoid exposing DB passwords in shell history and process lists.
- If you use env or flags modes, clean shell history and avoid sharing terminal logs.
- The generated command is created in the browser; secrets are not written into a static script on PMM Server.

## Troubleshooting

- If TLS is self-signed, enable `PMM_SERVER_INSECURE_TLS`.
- If package install fails, verify outbound access to Percona repositories.
- If registration fails for existing nodes, enable force re-registration (`PMM_REGISTER_FORCE=1` / `--register-force`).
