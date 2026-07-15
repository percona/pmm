# PMM + HolmesGPT (ADRE) — one-command deployment

`./install.sh` brings up PMM + HolmesGPT with HTTPS and auto-configures the AI backend. **No files to pre-place, no manual UI steps.**

## Prerequisites

A Linux host with **Docker + the Compose plugin** (`docker compose`), plus **`openssl`**, **`curl`**, and **`bash`**. `install.sh` runs `openssl` and the provisioning `curl`s directly on the host (not in throwaway containers). If your user needs `sudo` for Docker, the script auto-detects it (override with `DOCKER="sudo docker"`).

## Install

```bash
cp .env.example .env
# set: PMM_IMAGE (an ADRE build — see below), HOLMES_IMAGE (>= 0.36.0),
#      MODEL_NAME / LITELLM_MODEL / LLM_API_KEY
./install.sh
```

`install.sh` does four things: (1) generates a self-signed TLS cert into `$HOLMES_CONFIG_DIR/certs`
(host `openssl`); (2) `docker compose up -d pmm-server`; (3) provisions ADRE via the PMM API — adds
your model, enables ADRE, mints `PMM_API_TOKEN` + `HOLMES_API_KEY`, renders `.env` + config; (4)
`docker compose up -d holmesgpt`, which serves **HTTPS on 5050** natively. Open PMM at `https://<host>`
and the AI Assistant is ready.

It's re-runnable: the cert is kept if present and the provisioning calls are idempotent.

> The `holmesgpt` service is started **last, after** provisioning has minted the tokens and rendered the
> shared `.env`, so Holmes reads them on first boot — no restart needed. If you instead bring everything
> up at once (`docker compose up -d`), Holmes starts before the tokens exist and you must recreate it
> after provisioning: `docker compose up -d --force-recreate holmesgpt`.

## What you must provide (the only inputs)

- **`PMM_IMAGE`** — an ADRE-capable PMM Server image. ADRE (AI Deployment, reload, HTTPS enforcement) lives on
  the `tibi-holmes` fork; **the official `percona/pmm-server:3` does not have it** (`/v1/adre/*` → 404). Use the
  Feature Build image for that branch.
- **`HOLMES_IMAGE`** — `>= 0.36.0` (native HTTPS + `/api/admin/reload`); the official image is
  `robustadev/holmes:0.36.0` ([release](https://github.com/HolmesGPT/holmesgpt/releases/tag/0.36.0)). Pin a digest for prod.
- **Model + key** — `MODEL_NAME` (no slashes), `LITELLM_MODEL`, `LLM_API_KEY`.

Everything else (config dir, TLS cert, bootstrap `.env`, provisioning) is automated.

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

Edit in **PMM UI → AI Deployment** and **Save**. PMM renders to the shared dir and hot-reloads Holmes via
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
- **No `.py` patches, no init containers.** The old `database.py` / `openai_formatting.py` mounts were
  0.20/0.21 workarounds (strict-schema is now the `TOOL_SCHEMA_NO_PARAM_OBJECT_IF_NO_PARAMS` env var; the
  ClickHouse-HTTP one is an opt-in config flag). openssl + the provisioning curls run on the host, not in
  throwaway containers.
- **First PMM start can take a few minutes** (fresh `/srv` init); `install.sh` waits up to ~10 minutes for the
  ADRE API before giving up.
- **k8s toolset:** if you need it, add `- /path/to/kube_config:/root/.kube/config` to the `holmesgpt` volumes
  (omitted by default).
- **`sudo docker`:** `install.sh` auto-detects whether the daemon needs sudo; override with `DOCKER="sudo docker"`.
- **Single-host lab bundle — not hardened for shared hosts.** Change the Grafana admin password (`PMM_ADMIN_PASSWORD`, defaults to `admin`), and note that `install.sh` makes the shared config dir `0777` and that secrets (the minted PMM/Holmes tokens, the self-signed TLS key, and `model_list.yaml` with your LLM key) live in it. That's fine on a single-admin host; on a multi-user host, tighten those perms and prefer passing secrets via stdin rather than the provisioning args.

## Slack auto-investigate

The AI Assistant chat works after install. To also get **auto-investigation of firing alerts posted into the
alert's Slack thread**, configure the Slack bot + Grafana contact point — see
[../../../documentation/docs/use/ai-features/adre-slack-bot.md](../../../documentation/docs/use/ai-features/adre-slack-bot.md).
The key point: auto-investigate is triggered by the Slack **alert scrape**, which only acts on messages that
**@-mention the PMM bot** — so the Grafana Slack contact point's **"Mention Users"** must be set to the PMM bot's
user id. There is no webhook and no reconciliation poll; the mention scrape is the sole trigger.
