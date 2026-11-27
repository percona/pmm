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
	"testing"
	"time"

	"github.com/percona/pmm/managed/models"
)

func settingsWithAllow(channels, users []string) *models.Settings {
	s := &models.Settings{}
	s.Adre.SlackAllowedChannels = channels
	s.Adre.SlackAllowedUsers = users
	return s
}

func TestAuthorizeHuman(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		channels  []string
		users     []string
		ch, user  string
		wantAllow bool
	}{
		{"both allowed", []string{"C1"}, []string{"U1"}, "C1", "U1", true},
		{"channel only", []string{"C1"}, nil, "C1", "U1", false},
		{"user only", nil, []string{"U1"}, "C1", "U1", false},
		{"neither fail-closed", nil, nil, "C1", "U1", false},
		{"wrong channel", []string{"C2"}, []string{"U1"}, "C1", "U1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := authorizeHuman(settingsWithAllow(tt.channels, tt.users), tt.ch, tt.user)
			if d.allow != tt.wantAllow {
				t.Fatalf("allow=%v want %v", d.allow, tt.wantAllow)
			}
			if !d.allow && !d.notify {
				t.Fatal("a denied human interaction must set notify")
			}
		})
	}
}

func TestUserRateLimiterBurst(t *testing.T) {
	t.Parallel()
	rl := newUserRateLimiter(60, 2, 16) // 60/min, burst 2
	if !rl.allow("U1") {
		t.Fatal("first request should pass (burst 2)")
	}
	if !rl.allow("U1") {
		t.Fatal("second request should pass (burst 2)")
	}
	if rl.allow("U1") {
		t.Fatal("third immediate request should be throttled")
	}
	if !rl.allow("U2") {
		t.Fatal("a different user has its own bucket")
	}
}

func TestUserRateLimiterEviction(t *testing.T) {
	t.Parallel()
	rl := newUserRateLimiter(1, 1, 2)
	rl.allow("a")
	rl.allow("b")
	rl.allow("c") // exceeds cap → evicts oldest
	if len(rl.buckets) > 2 {
		t.Fatalf("expected map bounded to 2, got %d", len(rl.buckets))
	}
}

func TestCooldown(t *testing.T) {
	t.Parallel()
	c := newCooldown(time.Hour, 16)
	if !c.allow("k") {
		t.Fatal("first notice should be allowed")
	}
	if c.allow("k") {
		t.Fatal("second notice within window should be blocked")
	}
}
