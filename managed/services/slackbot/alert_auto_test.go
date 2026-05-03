// Copyright (C) 2026 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package slackbot

import (
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
)

func TestSlackMessagePlainBlob(t *testing.T) {
	t.Parallel()
	ev := &slackevents.MessageEvent{
		Text: "",
		Message: &slack.Msg{
			Attachments: []slack.Attachment{
				{Fallback: "[FIRING:1] HighCPU", Title: "Alert", Text: "instance=foo"},
			},
		},
	}
	got := slackMessagePlainBlob(ev)
	if !strings.Contains(strings.ToUpper(got), "FIRING") {
		t.Fatalf("expected FIRING in blob, got %q", got)
	}
	if !strings.Contains(got, "instance=foo") {
		t.Fatalf("expected attachment text, got %q", got)
	}
}

func TestSlackBotMessageSubtypeOK(t *testing.T) {
	t.Parallel()
	if !slackBotMessageSubtypeOK("") || !slackBotMessageSubtypeOK("bot_message") {
		t.Fatal("expected empty and bot_message to be ok")
	}
	if slackBotMessageSubtypeOK("message_changed") {
		t.Fatal("message_changed should not be ok")
	}
}
