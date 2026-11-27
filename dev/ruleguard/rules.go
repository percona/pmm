// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build ruleguard

// Package ruleguard holds custom rules for gocritic's ruleguard check.
package ruleguard

import "github.com/quasilyte/go-ruleguard/dsl"

// quoteVerb matches a %q verb, with optional flags, width or index, that isn't an escaped %%q.
const quoteVerb = `(^|[^%])(%%)*%[-+# 0-9.*\[\]]*q`

const quoteVerbMessage = `don't use %q in error and log messages; use %s, or '%s' when the value can contain spaces`

func quoteVerbInErrors(m dsl.Matcher) {
	m.Import("google.golang.org/grpc/status")

	m.Match(
		`fmt.Errorf($format, $*_)`,
		`status.Errorf($_, $format, $*_)`,
		`status.Newf($_, $format, $*_)`,
	).
		Where(m["format"].Text.Matches(quoteVerb)).
		Report(quoteVerbMessage).At(m["format"])
}

func quoteVerbInLogs(m dsl.Matcher) {
	m.Import("github.com/sirupsen/logrus")

	m.Match(
		`$log.Tracef($format, $*_)`,
		`$log.Debugf($format, $*_)`,
		`$log.Infof($format, $*_)`,
		`$log.Printf($format, $*_)`,
		`$log.Warnf($format, $*_)`,
		`$log.Warningf($format, $*_)`,
		`$log.Errorf($format, $*_)`,
		`$log.Fatalf($format, $*_)`,
		`$log.Panicf($format, $*_)`,
		`$log.Logf($_, $format, $*_)`,
	).
		Where(m["format"].Text.Matches(quoteVerb) && m["log"].Type.Implements("logrus.FieldLogger")).
		Report(quoteVerbMessage).At(m["format"])

	m.Match(
		`logrus.Tracef($format, $*_)`,
		`logrus.Debugf($format, $*_)`,
		`logrus.Infof($format, $*_)`,
		`logrus.Printf($format, $*_)`,
		`logrus.Warnf($format, $*_)`,
		`logrus.Warningf($format, $*_)`,
		`logrus.Errorf($format, $*_)`,
		`logrus.Fatalf($format, $*_)`,
		`logrus.Panicf($format, $*_)`,
	).
		Where(m["format"].Text.Matches(quoteVerb)).
		Report(quoteVerbMessage).At(m["format"])
}
