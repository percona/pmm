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
		Report(quoteVerbMessage)
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
	).
		Where(m["format"].Text.Matches(quoteVerb) && m["log"].Type.Implements("logrus.FieldLogger")).
		Report(quoteVerbMessage)

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
		Report(quoteVerbMessage)
}
