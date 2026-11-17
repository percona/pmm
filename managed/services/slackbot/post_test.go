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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type slackCall struct {
	method, text, blocks, threadTS string
}

// fakeSlack records chat.update and chat.postMessage calls; refuseBlocks makes it answer invalid_blocks to any call carrying blocks.
func fakeSlack(t *testing.T, refuseBlocks bool) (*slack.Client, func() []slackCall) {
	t.Helper()
	var mu sync.Mutex
	var calls []slackCall
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c := slackCall{
			method:   strings.TrimPrefix(r.URL.Path, "/"),
			text:     r.Form.Get("text"),
			blocks:   r.Form.Get("blocks"),
			threadTS: r.Form.Get("thread_ts"),
		}
		mu.Lock()
		calls = append(calls, c)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if refuseBlocks && c.blocks != "" {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_blocks"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"2.0"}`))
	}))
	t.Cleanup(srv.Close)
	return slack.New("xoxb-test", slack.OptionAPIURL(srv.URL+"/")), func() []slackCall {
		mu.Lock()
		defer mu.Unlock()
		return append([]slackCall(nil), calls...)
	}
}

const answerMD = "## Summary\n\nThis PMM has **14 nodes**.\n\n| Service | Node |\n|---|---|\n| ps-1 | node-1 |"

func TestPostAnswerAsMarkdownBlock(t *testing.T) {
	api, calls := fakeSlack(t, false)
	postAnswer(t.Context(), api, logrus.NewEntry(logrus.New()), "C1", "1.0", "1.5", answerMD)

	got := calls()
	require.Len(t, got, 1)
	assert.Equal(t, "chat.update", got[0].method)
	assert.Equal(t, "This PMM has 14 nodes.", got[0].text)
	var blocks []map[string]string
	require.NoError(t, json.Unmarshal([]byte(got[0].blocks), &blocks))
	assert.Equal(t, []map[string]string{{"type": "markdown", "text": answerMD}}, blocks)
}

func TestPostAnswerFallsBackToText(t *testing.T) {
	api, calls := fakeSlack(t, true)
	postAnswer(t.Context(), api, logrus.NewEntry(logrus.New()), "C1", "1.0", "1.5", answerMD)

	got := calls()
	require.Len(t, got, 2)
	assert.NotEmpty(t, got[0].blocks)
	assert.Equal(t, slackCall{
		method: "chat.update",
		text:   "*Summary*\n\nThis PMM has *14 nodes*.\n\n| Service | Node |\n|---|---|\n| ps-1 | node-1 |",
	}, got[1])
}

func TestPostMarkdownFallsBackToTextInThread(t *testing.T) {
	api, calls := fakeSlack(t, true)
	postMarkdown(t.Context(), logrus.NewEntry(logrus.New()), api, "C1", "1.0", "**Root cause:** disk full")

	got := calls()
	require.Len(t, got, 2)
	assert.Equal(t, "1.0", got[0].threadTS)
	assert.Equal(t, slackCall{method: "chat.postMessage", text: "*Root cause:* disk full", threadTS: "1.0"}, got[1])
}
