// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package slackbot

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/managed/models"
	"github.com/percona/pmm/managed/services/adre"
)

var mentionStripRE = regexp.MustCompile(`<@[^>]+>`)

// Run polls settings every 30s (and once at startup) and runs Slack Socket Mode while ADRE Slack is enabled.
func Run(ctx context.Context, db *reform.DB, l *logrus.Entry) error {
	ticker := time.NewTicker(30 * time.Second)
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
		fp := slackFingerprint(settings)

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
			runSocketMode(sctx, db, l)
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
func slackFingerprint(settings *models.Settings) string {
	if !settings.IsAdreEnabled() || !settings.Adre.SlackEnabled {
		return ""
	}
	u := strings.TrimSpace(settings.GetAdreURL())
	if u == "" {
		return ""
	}
	bot := strings.TrimSpace(settings.Adre.SlackBotToken)
	app := strings.TrimSpace(settings.Adre.SlackAppToken)
	if bot == "" || app == "" {
		return ""
	}
	return u + "\x00" + bot + "\x00" + app
}

func runSocketMode(ctx context.Context, db *reform.DB, l *logrus.Entry) {
	log := l.WithField("component", "adre-slack")
	settings, err := models.GetSettings(db)
	if err != nil {
		log.Errorf("GetSettings: %v", err)
		return
	}
	appTok := strings.TrimSpace(settings.Adre.SlackAppToken)
	api := slack.New(settings.Adre.SlackBotToken, slack.OptionAppLevelToken(appTok))
	sm := socketmode.New(api)

	auth, err := api.AuthTestContext(ctx)
	if err != nil {
		log.Errorf("AuthTest: %v", err)
		return
	}
	botUserID := auth.UserID

	ts := NewThreadStore()

	go func() {
		if err := sm.RunContext(ctx); err != nil && ctx.Err() == nil {
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
			sm.Ack(*evt.Request)
			if eventsAPI.Type != slackevents.CallbackEvent {
				continue
			}
			handleEventsAPI(ctx, db, api, ts, log, eventsAPI, botUserID)
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
		handleTurn(ctx, db, api, ts, log, eventsAPI.TeamID, ev.Channel, threadTS, ev.TimeStamp, text)

	case *slackevents.MessageEvent:
		if ev.BotID != "" && slackBotMessageSubtypeOK(ev.SubType) {
			settings, err := models.GetSettings(db)
			if err != nil {
				log.Errorf("GetSettings: %v", err)
				return
			}
			if settings.Adre.SlackAutoInvestigate &&
				settings.IsAdreEnabled() &&
				strings.TrimSpace(settings.GetAdreURL()) != "" {
				blob := slackMessagePlainBlob(ev)
				upper := strings.ToUpper(blob)
				if strings.Contains(upper, "FIRING") && !strings.Contains(upper, "RESOLVED") {
					threadTS := ev.ThreadTimeStamp
					if threadTS == "" {
						threadTS = ev.TimeStamp
					}
					handleTurn(ctx, db, api, ts, log, eventsAPI.TeamID, ev.Channel, threadTS, ev.TimeStamp, slackAutoInvestigatePrefix+blob)
					return
				}
			}
		}

		if ev.ThreadTimeStamp == "" {
			return
		}
		if ev.SubType != "" {
			return
		}
		if ev.BotID != "" || ev.User == "" {
			return
		}
		if ev.User == botUserID {
			return
		}
		text := strings.TrimSpace(ev.Text)
		handleTurn(ctx, db, api, ts, log, eventsAPI.TeamID, ev.Channel, ev.ThreadTimeStamp, ev.TimeStamp, text)

	default:
		return
	}
}

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
	teamID, channelID, threadTS, messageTS, userText string,
) {
	userText = strings.TrimSpace(userText)
	if userText == "" {
		return
	}

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

	if !slackEventDedupe.firstSeen(teamID, channelID, messageTS) {
		return
	}

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

	client := adre.NewClient(settings.GetAdreURL())

	doChat := func(extra string) (*adre.ChatResponse, error) {
		if err := acquireAdreChatSlot(ctx); err != nil {
			return nil, err
		}
		defer releaseAdreChatSlot()
		start := time.Now()
		req := adre.BuildSlackChatRequest(settings, userText, history, extra)
		resp, err := client.Chat(ctx, req)
		adreChatSeconds.Observe(time.Since(start).Seconds())
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
	if formatted == "" {
		formatted = " "
	}
	_, _, _, err = api.UpdateMessageContext(ctx, channelID, thinkingTS, slack.MsgOptionText(formatted, false))
	if err != nil {
		log.Warnf("UpdateMessage: %v", err)
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
