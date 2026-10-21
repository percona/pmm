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

package models

import "testing"

func TestSlackListContainsFailClosed(t *testing.T) {
	t.Parallel()
	if slackListContains(nil, "U1") {
		t.Error("empty list must contain nothing (fail-closed)")
	}
	if !slackListContains([]string{"U1", "U2"}, "U2") {
		t.Error("exact match should be found")
	}
	if !slackListContains([]string{" U1 "}, "U1") {
		t.Error("entries are trimmed")
	}
	if slackListContains([]string{"U1"}, "u1") {
		t.Error("match is case-sensitive (Slack IDs are uppercase)")
	}
	if slackListContains([]string{"U1"}, "") {
		t.Error("empty id never matches")
	}
}

func TestIsSlackAllowedHelpers(t *testing.T) {
	t.Parallel()
	s := &Settings{}
	s.Adre.SlackAllowedChannels = []string{"C1"}
	s.Adre.SlackAllowedUsers = []string{"U1"}
	if !s.IsSlackChannelAllowed("C1") || !s.IsSlackUserAllowed("U1") {
		t.Error("allow-listed channel/user should be allowed")
	}
	if s.IsSlackChannelAllowed("C2") || s.IsSlackUserAllowed("U2") {
		t.Error("non-listed channel/user should be denied")
	}
}

func TestNormalizeSlackIDs(t *testing.T) {
	t.Parallel()
	got := normalizeSlackIDs([]string{" U1 ", "", "U2", "U1"})
	if len(got) != 2 {
		t.Fatalf("expected trim+drop-empty+dedup to yield 2, got %d (%v)", len(got), got)
	}
}

func TestIsASCIIAlnum(t *testing.T) {
	t.Parallel()
	if !isASCIIAlnum("CABC123") {
		t.Error("alphanumeric should pass")
	}
	if isASCIIAlnum("") || isASCIIAlnum("C-123") || isASCIIAlnum("C 123") {
		t.Error("empty / hyphen / space should fail")
	}
}

func TestValidateSlackIDList(t *testing.T) {
	t.Parallel()
	if err := validateSlackIDList("f", []string{"C12345", "U67890"}); err != nil {
		t.Errorf("valid IDs should pass: %v", err)
	}
	if err := validateSlackIDList("f", []string{"short"}); err == nil {
		t.Error("too-short ID should fail")
	}
	if err := validateSlackIDList("f", []string{"bad-id-here"}); err == nil {
		t.Error("non-alphanumeric ID should fail")
	}
	big := make([]string, 201)
	for i := range big {
		big[i] = "C123456"
	}
	if err := validateSlackIDList("f", big); err == nil {
		t.Error("over 200 entries should fail")
	}
}

func TestValidateAdreSlackListParams(t *testing.T) {
	t.Parallel()
	sev := "critical"
	cap0 := -1
	if err := validateAdreSlackListParams(&ChangeSettingsParams{AutoInvestigateMinSeverity: &sev}); err != nil {
		t.Errorf("critical severity should be valid: %v", err)
	}
	bad := "huge"
	if err := validateAdreSlackListParams(&ChangeSettingsParams{AutoInvestigateMinSeverity: &bad}); err == nil {
		t.Error("unknown severity should fail")
	}
	if err := validateAdreSlackListParams(&ChangeSettingsParams{AutoInvestigateHourlyCap: &cap0}); err == nil {
		t.Error("negative hourly cap should fail")
	}
	matchers := []string{"no-equals-sign"}
	if err := validateAdreSlackListParams(&ChangeSettingsParams{AutoInvestigateLabelMatchers: &matchers}); err == nil {
		t.Error("label matcher without = should fail")
	}
}
