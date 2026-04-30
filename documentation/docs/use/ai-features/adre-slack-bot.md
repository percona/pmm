# ADRE Slack bot (PMM-managed)

The Slack integration runs inside **pmm-managed** on the **HA leader** only. It uses **Socket Mode** (no public HTTP endpoint for Slack events).

## Configuration

In **PMM → Configuration → AI Assistant**, enable **Slack** and paste the bot and Socket Mode tokens.

For **clickable** `/v1/grafana/render/blob/...` links in Slack text, set **Public address** on **PMM Settings → Advanced** (the same value used elsewhere in PMM). The Slack bot uses that origin when rewriting relative PMM URLs.

## Slack app (operator)

- **Bot User OAuth token** (`xoxb-`) and **App-Level Token** with `connections:write` (`xapp-`) are stored in PMM settings (not returned on GET).
- **OAuth scopes (bot)** typically include: `app_mentions:read`, `chat:write`, `files:write`, and channel/group history scopes as needed for thread replies.
- **Event subscriptions** (delivered over Socket Mode): at minimum `app_mention` and `message` for the channels you use.

## Migration from a standalone bot

1. Configure PMM Slack settings and verify mentions work.
2. **Stop** any external Holmes/PMM Slack bot process so only one Socket Mode client connects; duplicate bots cause duplicate answers.

## HA behavior

Only the **Raft leader** opens Socket Mode. After failover, **in-thread Slack context is not preserved** on the new leader (in-memory only for this release).
