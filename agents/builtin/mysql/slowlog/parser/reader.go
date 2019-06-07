// pmm-agent
// Copyright (C) 2018 Percona LLC
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

package parser

// ReaderMetrics contains Reader metrics.
type ReaderMetrics struct {
	InputSize int64
	InputPos  int64
}

// A Reader reads lines from the underlying source.
//
// Implementation should allow concurrent calls to different methods.
type Reader interface {
	// NextLine reads full lines from the underlying source and returns them (including the last '\n').
	// If the full line can't be read because of EOF, reader implementation may decide to return it,
	// or block and wait for new data to arrive. Other errors should be returned without blocking.
	// NextLine also should not block when the source is closed, but it may return buffered data while it has it.
	NextLine() (string, error)

	// Close closes the underlying source. A caller should continue to call NextLine until error is returned.
	Close() error

	// Metrics returns current metrics.
	Metrics() *ReaderMetrics
}
