// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package slackbot

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
	"github.com/percona/pmm/managed/services/autoinvestigate"
)

// AlertProcessor triggers auto-investigations from scraped Slack alert messages. *autoinvestigate.Service
// implements it; it is the same entry point the webhook/poll fallback uses, so the scrape inherits the
// selection guards, episode dedup, hourly cap and idempotent claim.
type AlertProcessor interface {
	ProcessAlerts(ctx context.Context, alerts []autoinvestigate.Alert)
}

var mentionStripRE = regexp.MustCompile(`<@[^>]+>`)

// slackTurnTimeout bounds a single Slack turn (investigation + posting). It must exceed the Holmes
// client timeout (adre streamTimeout, 5m) so a completed investigation can still be posted back.
const slackTurnTimeout = 10 * time.Minute

// Run polls settings every 30s (and once at startup) and runs Slack Socket Mode while ADRE Slack is enabled.
func Run(ctx context.Context, db *reform.DB, processor AlertProcessor, l *logrus.Entry) error {
	ticker := time.NewTicker(30 * time.Second) //nolint:mnd
	defer ticker.Stop()

	var mu sync.Mutex
	var sockCancel context.CancelFunc
	var spawnSeq atomic.Uint64
	lastFP := ""

	stopEverything := func() {
		spawnSeq.Add(1)
		if sockCancel != nil {
			sockCancel()
			sockCancel = nil
		}
		lastFP = ""
		slackConnected.Set(0)
	}

	defer func() {
		mu.Lock()
		stopEverything()
		mu.Unlock()
	}()

	eval := func() {
		settings, err := models.GetSettings(db)
		if err != nil {
			l.Errorf("GetSettings: %v", err)
			return
		}
		prov, err := models.GetAdreProvisioning(db)
		if err != nil {
			l.Errorf("GetAdreProvisioning: %v", err)
			return
		}
		fp := slackFingerprint(settings, prov.SlackBotToken, prov.SlackAppToken)

		mu.Lock()
		if fp == "" {
			stopEverything()
			mu.Unlock()
			return
		}
		if fp == lastFP && sockCancel != nil {
			mu.Unlock()
			return
		}

		spawnSeq.Add(1)
		if sockCancel != nil {
			sockCancel()
			sockCancel = nil
		}
		lastFP = fp

		sctx, cancel := context.WithCancel(ctx)
		sockCancel = cancel
		gen := spawnSeq.Load()

		mu.Unlock()

		slackConnected.Set(1)
		go func(runGen uint64) {
			runSocketMode(sctx, db, processor, l)
			mu.Lock()
			defer mu.Unlock()
			if runGen != spawnSeq.Load() {
				return
			}
			sockCancel = nil
			lastFP = ""
			slackConnected.Set(0)
		}(gen)
	}

	eval()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			eval()
		}
	}
}

// slackFingerprint identifies a Socket Mode session for hot-reload. Must never be logged (contains tokens).
// The bot/app tokens are stored encrypted in adre_provisioning and passed in decrypted by the caller.
func slackFingerprint(settings *models.Settings, botToken, appToken string) string {
	if !settings.IsAdreEnabled() || !settings.Adre.SlackEnabled {
		return ""
	}
	u := strings.TrimSpace(settings.GetAdreURL())
	if u == "" {
		return ""
	}
	bot := strings.TrimSpace(botToken)
	app := strings.TrimSpace(appToken)
	if bot == "" || app == "" {
		return ""
	}
	return u + "\x00" + bot + "\x00" + app
}

func runSocketMode(ctx context.Context, db *reform.DB, processor AlertProcessor, l *logrus.Entry) {
	log := l.WithField("component", "adre-slack")
	prov, err := models.GetAdreProvisioning(db)
	if err != nil {
		log.Errorf("GetAdreProvisioning: %v", err)
		return
	}
	appTok := strings.TrimSpace(prov.SlackAppToken)
	api := slack.New(prov.SlackBotToken, slack.OptionAppLevelToken(appTok))
	sm := socketmode.New(api)

	auth, err := api.AuthTestContext(ctx)
	if err != nil {
		log.Errorf("AuthTest: %v", err)
		return
	}
	botUserID := auth.UserID
	selfBotID := auth.BotID // this bot's own bot_id, used to skip scraping the bot's own posts

	ts := NewThreadStore()

	go func() {
		err := sm.RunContext(ctx)
		if err != nil && ctx.Err() == nil {
			log.Warnf("socket mode ended: %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-sm.Events:
			if !ok {
				return
			}
			if evt.Type != socketmode.EventTypeEventsAPI || evt.Request == nil {
				continue
			}
			eventsAPI, ok := evt.Data.(slackevents.EventsAPIEvent)
			if !ok {
				continue
			}
			_ = sm.Ack(*evt.Request)
			if eventsAPI.Type != slackevents.CallbackEvent {
				continue
			}
			// Handle in a goroutine so a long investigation does not block the event loop (which
			// would freeze the bot for every channel). Per-thread serialization and the chat-slot
			// semaphore in handleTurn keep concurrency safe.
			go handleEventsAPI(ctx, db, api, ts, log, eventsAPI, botUserID, selfBotID, processor)
		}
	}
}

func handleEventsAPI(
	ctx context.Context,
	db *reform.DB,
	api *slack.Client,
	ts *ThreadStore,
	log *logrus.Entry,
	eventsAPI slackevents.EventsAPIEvent,
	botUserID string,
	selfBotID string,
	processor AlertProcessor,
) {
	switch ev := eventsAPI.InnerEvent.Data.(type) {
	case *slackevents.AppMentionEvent:
		if ev.BotID != "" {
			return
		}
		threadTS := ev.ThreadTimeStamp
		if threadTS == "" {
			threadTS = ev.TimeStamp
		}
		text := stripMentions(ev.Text)
		gateAndHandle(ctx, db, api, ts, log, eventsAPI.TeamID, ev.Channel, threadTS, ev.TimeStamp, ev.User, triggerMention, text)

	case *slackevents.MessageEvent:
		// Alert scrape: a Grafana alert message (a bot message in a configured alert channel) triggers a
		// persisted, fingerprint-keyed auto-investigation posted back in this message's thread. Bot
		// messages never fall through to human-chat handling. Skip our own posts (started/report notices
		// land in the same alert channel) so the bot can never scrape itself into a loop.
		if ev.BotID != "" {
			if ev.BotID != selfBotID && slackBotMessageSubtypeOK(ev.SubType) {
				handleAlertScrape(ctx, db, log, eventsAPI.TeamID, ev, processor)
			}
			return
		}
		// Human thread reply. (Auto-investigate is driven by the scrape above / Grafana Alertmanager.)
		if ev.ThreadTimeStamp == "" {
			return
		}
		if ev.SubType != "" {
			return
		}
		if ev.User == "" || ev.User == botUserID {
			return
		}
		text := strings.TrimSpace(ev.Text)
		gateAndHandle(ctx, db, api, ts, log, eventsAPI.TeamID, ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, ev.User, triggerThread, text)

	default:
		return
	}
}

// handleAlertScrape parses a Grafana alert message and, when it comes from a configured alert channel
// (and bot, if SlackAlertBotIDs is set), runs it through the persistent auto-investigate pipeline with a
// Slack thread ref so the started notice and report post back as replies under this alert.
func handleAlertScrape(ctx context.Context, db *reform.DB, log *logrus.Entry, teamID string, ev *slackevents.MessageEvent, processor AlertProcessor) {
	if processor == nil {
		return
	}
	settings, err := models.GetSettings(db)
	if err != nil {
		log.Errorf("GetSettings: %v", err)
		return
	}
	if !isAlertSource(settings, ev.Channel, ev.BotID) {
		return
	}
	alert, ok := parseSlackAlert(ev)
	if !ok {
		return
	}
	threadTS := ev.ThreadTimeStamp
	if threadTS == "" {
		threadTS = ev.TimeStamp
	}
	alert.Slack = &autoinvestigate.SlackRef{TeamID: teamID, Channel: ev.Channel, ThreadTS: threadTS}
	processor.ProcessAlerts(ctx, []autoinvestigate.Alert{alert})
}

// gateAndHandle authorizes and rate-limits a human Slack interaction before running a turn. Denied
// interactions get one cooled-down "ask an admin" reply; throttled ones get one cooled-down "too
// fast" reply. Both keep the bot from being weaponized into a reply-spam amplifier.
func gateAndHandle( //nolint:gocognit
	ctx context.Context,
	db *reform.DB,
	api *slack.Client,
	ts *ThreadStore,
	log *logrus.Entry,
	teamID, channelID, threadTS, messageTS, userID string,
	trigger slackTrigger,
	text string,
) {
	settings, err := models.GetSettings(db)
	if err != nil {
		log.Errorf("GetSettings: %v", err)
		return
	}
	if !settings.IsAdreEnabled() || strings.TrimSpace(settings.GetAdreURL()) == "" {
		return
	}

	// Drop Slack event redeliveries first, so duplicates never consume a user's rate-limit budget or
	// trigger a duplicate denial reply. handleTurn forgets the key if it fails before answering.
	if !slackEventDedupe.firstSeen(teamID, channelID, messageTS) {
		return
	}

	if d := authorizeHuman(settings, channelID, userID); !d.allow {
		slackAuthzDeniedTotal.WithLabelValues(string(trigger)).Inc()
		if d.notify && denyCooldown.allow(channelID+"\x00"+userID) {
			postThreadLine(ctx, log, api, channelID, threadTS, slackDenyMsg)
		}
		return
	}

	if !slackUserLimiter.allow(userID) {
		slackRateLimitedTotal.Inc()
		if denyCooldown.allow("rl\x00" + channelID + "\x00" + userID) {
			postThreadLine(ctx, log, api, channelID, threadTS, slackRateLimitMsg)
		}
		return
	}

	handleTurn(ctx, db, api, ts, log, teamID, channelID, threadTS, messageTS, userID, trigger, text)
}

const (
	slackDenyMsg = "I'm not enabled for this channel/user yet. Ask a PMM admin to add it under " +
		"Settings → AI Assistant → Slack."
	slackRateLimitMsg = "You're sending requests too fast — please wait a moment and try again."
)

func stripMentions(s string) string {
	return strings.TrimSpace(mentionStripRE.ReplaceAllString(s, ""))
}

var adreChatSem = make(chan struct{}, maxConcurrentAdreChatSlots)

func postThreadLine(ctx context.Context, log *logrus.Entry, api *slack.Client, channelID, threadTS, msg string) {
	_, _, err := api.PostMessageContext(
		ctx, channelID,
		slack.MsgOptionText(msg, false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		log.Debugf("Slack thread reply: %v", err)
	}
}

func handleTurn(
	ctx context.Context,
	db *reform.DB,
	api *slack.Client,
	store *ThreadStore,
	log *logrus.Entry,
	teamID, channelID, threadTS, messageTS, invoker string,
	trigger slackTrigger,
	userText string,
) {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return
	}
	if len(userText) > maxSlackUserTextBytes {
		// Back off to a rune boundary so truncation never produces invalid UTF-8 mid-rune.
		cut := maxSlackUserTextBytes
		for cut > 0 && !utf8.RuneStart(userText[cut]) {
			cut--
		}
		userText = userText[:cut] + "\n…[truncated]"
	}

	// Detach from the socket/supervisor context so a websocket reconnect or settings reload mid-turn
	// cannot cancel the Holmes call or the final Slack post; bound the whole turn with a timeout.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), slackTurnTimeout)
	defer cancel()

	settings, err := models.GetSettings(db)
	if err != nil {
		log.Errorf("GetSettings: %v", err)
		postThreadLine(ctx, log, api, channelID, threadTS, "Could not load PMM settings. Try again later.")
		return
	}
	if !settings.IsAdreEnabled() || strings.TrimSpace(settings.GetAdreURL()) == "" {
		postThreadLine(ctx, log, api, channelID, threadTS, "AI Assistant is disabled or the assistant URL is not set.")
		return
	}

	// Event de-duplication already happened in gateAndHandle (before authz/rate-limit); handleTurn
	// only forgets the key on an early post failure so Slack's retry can re-process.

	// Audit who/where the turn came from — never the message body (may carry sensitive content).
	log.WithFields(logrus.Fields{
		"team": teamID, "channel": channelID, "thread": threadTS,
		"msg_ts": messageTS, "invoker": invoker, "trigger": string(trigger),
		"text_len": len(userText),
	}).Info("adre slack turn")

	key := ThreadKey{TeamID: teamID, ChannelID: channelID, ThreadTS: threadTS}
	unlock := acquireThreadLock(key)
	defer unlock()

	store.AppendUser(key, userText)
	history := store.Snapshot(key)

	_, thinkingTS, err := api.PostMessageContext(
		ctx, channelID,
		slack.MsgOptionText("Thinking…", false),
		slack.MsgOptionTS(threadTS),
	)
	if err != nil {
		log.Warnf("PostMessage: %v", err)
		store.UndoLastUserMessage(key, userText)
		slackEventDedupe.forget(teamID, channelID, messageTS)
		return
	}

	slackEventsTotal.Inc()

	client := adre.NewClientFromSettings(settings)

	doChat := func(extra string) (*adre.ChatResponse, error) {
		if err := acquireAdreChatSlot(ctx); err != nil { //nolint:noinlineerr
			return nil, err
		}
		defer releaseAdreChatSlot()
		start := time.Now()
		req := adre.BuildSlackChatRequest(settings, userText, history, extra)
		resp, err := client.Chat(ctx, req)
		latencyMs := int(time.Since(start).Milliseconds())
		adreChatSeconds.Observe(time.Since(start).Seconds())
		if err == nil && resp != nil {
			featureRef := teamID + ":" + channelID + ":" + threadTS
			_, _ = adre.RecordHolmesUsage(ctx, adre.UsageRecordInput{
				DB:          db,
				Feature:     adre.HolmesFeatureSlackChat,
				FeatureRef:  featureRef,
				Model:       req.Model,
				Metadata:    resp.Metadata,
				TriggeredBy: "slack-" + string(trigger),
				LatencyMs:   latencyMs,
			})
		}
		return resp, err
	}

	resp, err := doChat("")
	if err != nil {
		adreChatErrorsTotal.Inc()
		log.Warnf("Slack ADRE chat: %v", err)
		_, _, _, _ = api.UpdateMessageContext(ctx, channelID, thinkingTS, slack.MsgOptionText("Chat request failed; check PMM logs.", false))
		return
	}
	analysis := resp.Analysis

	if NeedsGraphRetry(userText, analysis) {
		resp2, err := doChat(graphRetryPrompt)
		if err != nil {
			adreChatErrorsTotal.Inc()
			log.Warnf("Slack ADRE chat (graph retry): %v", err)
		} else if resp2 != nil && strings.TrimSpace(resp2.Analysis) != "" {
			analysis = resp2.Analysis
		}
	}

	store.AppendAssistant(key, analysis)

	hashes := ExtractBlobHashes(analysis)
	uploaded := 0
	for _, h := range hashes {
		png, err := LoadBlobPNG(h)
		if err != nil {
			log.Debugf("LoadBlobPNG %s: %v", h, err)
			continue
		}
		_, err = api.UploadFileContext(ctx, slack.UploadFileParameters{
			Filename:        h + ".png",
			FileSize:        len(png),
			Reader:          bytes.NewReader(png),
			Channel:         channelID,
			ThreadTimestamp: threadTS,
			Title:           "Panel",
		})
		if err != nil {
			log.Warnf("UploadFile: %v", err)
			continue
		}
		uploaded++
		slackUploadsTotal.Inc()
	}

	hideLinks := len(hashes) > 0 && uploaded == len(hashes)
	formatted := FormatAnswerForSlack(analysis, settings.GetEffectiveSlackLinkBaseURL(), hideLinks)
	postAnswer(ctx, api, log, channelID, threadTS, thinkingTS, formatted)
}

// postAnswer replaces the "Thinking…" placeholder with the answer. Slack rejects over-long messages
// (msg_too_long), so long investigation reports are split into chunks: the first edits the
// placeholder, the rest are posted as threaded replies. If the edit fails for any reason, the first
// chunk is re-posted as a fresh reply so the result is never silently dropped (stuck on "Thinking…").
func postAnswer(ctx context.Context, api *slack.Client, log *logrus.Entry, channelID, threadTS, thinkingTS, answer string) {
	chunks := chunkForSlack(answer)
	if len(chunks) == 0 {
		chunks = []string{" "}
	}
	//nolint:dogsled // slack-go returns (channelID, ts, text, error); we only need err
	_, _, _, err := api.UpdateMessageContext(ctx, channelID, thinkingTS, slack.MsgOptionText(chunks[0], false))
	if err != nil {
		log.Warnf("UpdateMessage: %v", err)
		postThreadLine(ctx, log, api, channelID, threadTS, chunks[0])
	}
	for _, c := range chunks[1:] {
		postThreadLine(ctx, log, api, channelID, threadTS, c)
	}
}

func acquireAdreChatSlot(ctx context.Context) error {
	select {
	case adreChatSem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func releaseAdreChatSlot() {
	<-adreChatSem
}
