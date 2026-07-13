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

package adre

import (
	"testing"

	"github.com/percona/pmm/managed/models"
)

func TestNormalizeBehaviorControlsForHolmes_legacyTimeRunbooks(t *testing.T) {
	in := map[string]bool{
		"time_runbooks":          false,
		"todowrite_instructions": false,
	}
	out := NormalizeBehaviorControlsForHolmes(in)
	if _, ok := out["time_runbooks"]; ok {
		t.Fatalf("time_runbooks should be removed after normalize, got %#v", out)
	}
	if v, ok := out["time_skills"]; !ok || v != false {
		t.Fatalf("expected time_skills=false, got %#v", out)
	}
	if v, ok := out["todowrite_instructions"]; !ok || v != false {
		t.Fatalf("expected todowrite_instructions preserved, got %#v", out)
	}
}

func TestNormalizeBehaviorControlsForHolmes_prefersTimeSkills(t *testing.T) {
	in := map[string]bool{
		"time_runbooks": true,
		"time_skills":   false,
	}
	out := NormalizeBehaviorControlsForHolmes(in)
	if _, ok := out["time_runbooks"]; ok {
		t.Fatalf("time_runbooks should be removed, got %#v", out)
	}
	if v := out["time_skills"]; v != false {
		t.Fatalf("existing time_skills should win over time_runbooks, got time_skills=%v", v)
	}
}

func TestNormalizeBehaviorControlsForHolmes_nilEmpty(t *testing.T) {
	if NormalizeBehaviorControlsForHolmes(nil) != nil {
		t.Fatal("nil in should be nil out")
	}
	empty := map[string]bool{}
	out := NormalizeBehaviorControlsForHolmes(empty)
	if out == nil || len(out) != 0 {
		t.Fatalf("empty map: expected empty map out, got %#v", out)
	}
}

func TestDefaultBehaviorControlsFast_usesTimeSkills(t *testing.T) {
	m := DefaultBehaviorControlsFast()
	if _, ok := m["time_skills"]; !ok {
		t.Fatalf("expected time_skills in defaults, got %#v", m)
	}
	if _, ok := m["time_runbooks"]; ok {
		t.Fatalf("defaults must not use legacy time_runbooks, got %#v", m)
	}
}

func TestResolveBehaviorControlsForPostChat_normalizesLegacy(t *testing.T) {
	s := minimalSettingsWithAdre()
	s.Adre.BehaviorControlsFast = map[string]bool{
		"time_runbooks":          true,
		"todowrite_instructions": false,
	}
	out := ResolveBehaviorControlsForPostChat(s, "fast")
	if _, ok := out["time_runbooks"]; ok {
		t.Fatalf("resolved map should not contain time_runbooks, got %#v", out)
	}
	if v, ok := out["time_skills"]; !ok || v != true {
		t.Fatalf("expected time_skills=true from legacy, got %#v", out)
	}
}

func minimalSettingsWithAdre() *models.Settings {
	return &models.Settings{}
}
