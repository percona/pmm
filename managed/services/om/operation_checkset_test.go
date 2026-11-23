// Copyright (C) 2023 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package om

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeReportDigest(t *testing.T) {
	t.Parallel()
	set := CheckSet{Facts: []CheckFact{{Name: "fcv"}, {Name: "members"}}}

	first, err := computeReportDigest(set, map[string]any{"fcv": "7.0", "members": 3})
	require.NoError(t, err)
	same, err := computeReportDigest(set, map[string]any{"members": 3, "fcv": "7.0"})
	require.NoError(t, err)
	assert.Equal(t, first, same, "the digest is order-independent")

	changed, err := computeReportDigest(set, map[string]any{"fcv": "8.0", "members": 3})
	require.NoError(t, err)
	assert.NotEqual(t, first, changed, "a changed fact changes the digest")
}

func TestComputeReportDigestRejectsSecret(t *testing.T) {
	t.Parallel()
	set := CheckSet{Facts: []CheckFact{{Name: "fcv"}, {Name: "pbm_uri", Secret: true}}}

	_, err := computeReportDigest(set, map[string]any{"fcv": "7.0", "pbm_uri": "mongodb://secret"})
	require.Error(t, err)

	ok, err := computeReportDigest(set, map[string]any{"fcv": "7.0"})
	require.NoError(t, err)
	assert.NotEmpty(t, ok)
}

func TestRunBlockingRules(t *testing.T) {
	t.Parallel()
	pass := func(CheckContext) RuleResult { return RuleResult{} }
	block := func(CheckContext) RuleResult { return RuleResult{Blocked: true, Reason: "not_ready", Path: "members"} }

	result := runBlockingRules(CheckSet{Rules: []BlockingRule{pass, pass}}, CheckContext{})
	assert.False(t, result.Blocked)

	result = runBlockingRules(CheckSet{Rules: []BlockingRule{pass, block, pass}}, CheckContext{})
	require.True(t, result.Blocked)
	assert.Equal(t, "not_ready", result.Reason)
}

func TestReportStale(t *testing.T) {
	t.Parallel()
	set := CheckSet{Facts: []CheckFact{{Name: "fcv", MaxAge: time.Minute}}}
	facts := map[string]any{"fcv": "7.0"}
	digest, err := computeReportDigest(set, facts)
	require.NoError(t, err)

	fresh, err := reportStale(set, digest, facts, map[string]time.Duration{"fcv": time.Second})
	require.NoError(t, err)
	assert.False(t, fresh)

	moved, err := reportStale(set, digest, map[string]any{"fcv": "8.0"}, map[string]time.Duration{"fcv": time.Second})
	require.NoError(t, err)
	assert.True(t, moved, "a changed fact is stale")

	overAge, err := reportStale(set, digest, facts, map[string]time.Duration{"fcv": time.Hour})
	require.NoError(t, err)
	assert.True(t, overAge, "an over-age fact is stale")
}

type fakeScanRequester struct {
	requests []string
}

func (f *fakeScanRequester) RequestScan(operationID, profile, idempotencyKey string) error {
	f.requests = append(f.requests, operationID+"/"+profile+"/"+idempotencyKey)
	return nil
}

func TestScanRequesterFake(t *testing.T) {
	t.Parallel()
	var requester ScanRequester = &fakeScanRequester{}
	require.NoError(t, requester.RequestScan("op-1", "install_check", "key-1"))
	fake, ok := requester.(*fakeScanRequester)
	require.True(t, ok)
	assert.Equal(t, []string{"op-1/install_check/key-1"}, fake.requests)
}
