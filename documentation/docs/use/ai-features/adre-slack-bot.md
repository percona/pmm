# ADRE Slack bot (PMM-managed)

The Slack integration runs inside **pmm-managed** on the **HA leader** only. It uses **Socket Mode** (no public HTTP endpoint for Slack events).

## Configuration

In **PMM → Configuration → AI Assistant**, enable **Slack** and paste the bot and Socket Mode tokens.

For **clickable** `/v1/grafana/render/blob/...` links in Slack text, set **Public address** on **PMM Settings → Advanced** (the same value used elsewhere in PMM). The Slack bot uses that origin when rewriting relative PMM URLs.

## Slack app (operator)

- **Bot User OAuth token** (`xoxb-`) and **App-Level Token** with `connections:write` (`xapp-`) are stored in PMM settings (not returned on GET).
- **OAuth scopes (bot)** typically include: `app_mentions:read`, `chat:write`, `files:write`, and channel/group history scopes as needed for thread replies.
- **Event subscriptions** (delivered over Socket Mode): at minimum `app_mention` and `message` for the channels you use.

## Auto-investigate firing alerts

With **Auto-investigate firing alerts** on, the bot starts an investigation for each Grafana alert message posted in the **Alert channels** and posts the report in the alert thread. It acts only on bot messages that @-mention the PMM bot, so its own reports never trigger it.

Set up Grafana to post those messages:

1. Find the PMM bot's Slack member ID: pmm-managed logs it as `bot_user_id` when the bot connects (`grep bot_user_id /srv/logs/pmm-managed.log` in the PMM Server container).
2. In **Alerting → Contact points**, add a **Slack** contact point:
    - **Recipient**: an alert channel ID, for example `C0123456789`.
    - **Token**: a bot token that can post there. The PMM bot's own token works.
    - **Mention Users** (under **Optional Slack settings**): the PMM bot's member ID.
3. In **Alerting → Notification policies**, send alerts to that contact point. Set **Group by** to `...` so each alert gets its own message: the bot reads the labels of the first alert in a message only.
4. Select **Test** on the contact point. A `TestAlert` investigation starts and its notice appears in the message thread.

The severity floor, label matchers and hourly cap on the Assistant configuration page limit how many investigations run.

## Migration from a standalone bot

1. Configure PMM Slack settings and verify mentions work.
2. **Stop** any external Holmes/PMM Slack bot process so only one Socket Mode client connects; duplicate bots cause duplicate answers.

## HA behavior

Only the **Raft leader** opens Socket Mode. After failover, **in-thread Slack context is not preserved** on the new leader (in-memory only for this release).
