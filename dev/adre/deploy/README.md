# PMM + HolmesGPT (ADRE) — one-command deployment

`install.sh` brings up PMM + HolmesGPT with HTTPS and wires them together. Nothing has to be prepared or
edited first.

## Prerequisites

A Linux host with **Docker + the Compose plugin** (`docker compose`), **`openssl`**, **`curl`** and **`bash`**,
with port **443** free. If your user needs `sudo` for Docker, the script detects it.

## Install

```bash
mkdir -p ~/pmm-adre && cd ~/pmm-adre
curl -fsSL --remote-name-all https://raw.githubusercontent.com/percona/pmm/tibi-holmes/dev/adre/deploy/{compose.yaml,install.sh}
bash install.sh
```

It prints the PMM URL and the generated `admin` password (kept in `./.env`). Then open
**AI Assistant Deployment → Models & Keys**, add your model (name, LiteLLM model, API key) and **Save** —
HolmesGPT picks it up without a restart.

To add the model during install instead, pass it on the command line:

```bash
MODEL_NAME=claude-opus LITELLM_MODEL=anthropic/claude-opus-5-5 LLM_API_KEY=sk-ant-... bash install.sh
```

`install.sh`, in order: generates a self-signed TLS cert for HolmesGPT; starts `pmm-server` and
`grafana-image-renderer` and sets the admin password; picks the PMM URL HolmesGPT calls back on
(`https://<host IP>` when containers can reach it, else `https://pmm-server:8443`); enables ADRE and mints
`PMM_API_TOKEN` + `HOLMES_API_KEY` through the PMM API; starts `holmesgpt` (HTTPS on 5050) and checks that
PMM can reach it. Re-running it is safe: the cert, password and tokens are kept and models are upserted.

> Start Holmes through `install.sh` (or recreate it after provisioning with
> `docker compose up -d --force-recreate holmesgpt`): a plain `docker compose up -d` before provisioning
> starts Holmes without the tokens.

## Settings (all optional)

Defaults are in [`.env.example`](.env.example); override them in `./.env` or on the command line.

- **`PMM_IMAGE`** — an ADRE-capable PMM Server image (a Feature Build of `tibi-holmes`); **the official
  `percona/pmm-server:3` does not have ADRE** (`/v1/adre/*` → 404).
- **`HOLMES_IMAGE`** — `>= 0.36.0` (native HTTPS + `/api/admin/reload`); defaults to
  `robustadev/holmes:0.45.0` ([release](https://github.com/HolmesGPT/holmesgpt/releases/tag/0.45.0)). Pin a digest
  for prod. Since 0.43.0 Holmes turns TodoWrite planning off by default; PMM opts Investigation mode back in via
  `behavior_controls`.
- **`PMM_ADMIN_PASSWORD`**, **`PMM_CALLBACK_URL`**, **`HOLMES_CONFIG_DIR`** — see `.env.example`.
- **`MODEL_NAME` / `LITELLM_MODEL` / `LLM_API_KEY`** — all three or none.
- **`FAST_MODEL_NAME` / `FAST_LITELLM_MODEL`** (+ optional `FAST_LLM_API_KEY`) — a cheaper model for Fast chat,
  the default mode; `MODEL_NAME` then runs Investigation mode only. Every request carries roughly 25–30k tokens
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

Edit in **PMM UI → AI Assistant Deployment** and **Save**. PMM renders to the shared dir and hot-reloads Holmes via
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
- **Updates are disabled** (`PMM_ENABLE_UPDATES=false`): the in-app updater would replace this ADRE build with a
  stock PMM release, which has no ADRE. Move to a newer build by changing `PMM_IMAGE` and re-running `install.sh`.
- **k8s toolset:** if you need it, add `- /path/to/kube_config:/root/.kube/config` to the `holmesgpt` volumes
  (omitted by default).
- **Single-host lab bundle — not hardened for shared hosts.** The admin password is generated and stored in `./.env` (0600), and note that `install.sh` makes the shared config dir `0777` and that secrets (the minted PMM/Holmes tokens, the self-signed TLS key, and `model_list.yaml` with your LLM key) live in it. That's fine on a single-admin host; on a multi-user host, tighten those perms. `install.sh` sends secrets to the PMM API on stdin, never as command-line arguments.

## Slack auto-investigate

The AI Assistant chat works after install. To also get **auto-investigation of firing alerts posted into the
alert's Slack thread**, configure the Slack bot + Grafana contact point — see
[../../../documentation/docs/use/ai-features/adre-slack-bot.md](../../../documentation/docs/use/ai-features/adre-slack-bot.md).
The key point: auto-investigate is triggered by the Slack **alert scrape**, which only acts on messages that
**@-mention the PMM bot** — so the Grafana Slack contact point's **"Mention Users"** must be set to the PMM bot's
user id. There is no webhook and no reconciliation poll; the mention scrape is the sole trigger.
