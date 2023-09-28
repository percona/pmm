// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//  http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package progress contains progress bar programs to be rendered with BubbleTea.
package progress

import (
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/percona/pmm/admin/pkg/bubbles/styles"
)

const (
	padding  = 4
	maxWidth = 80
)

// SizeProperties holds properties of a progress bar.
type SizeProperties struct {
	ID      string
	Current int
	Prefix  string // Prepended to progress bar
	Total   int
	Suffix  string // Appended after progress bar
}

// UpdateProgressMsg is a message sent to program to indicate a change in progress.
type UpdateProgressMsg struct {
	SizeProperties
}

type size struct {
	progress progress.Model
	SizeProperties
}

// SizeModel represents multiple progress bars rendered one after another.
type SizeModel struct {
	progressBars []size
	Quitting     bool // Quitting is set when ctrl+c is received
}

// Init is run when the program starts.
func (m SizeModel) Init() tea.Cmd {
	return nil
}

// Update receives a message from program to be processed.
func (m SizeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if keypress := msg.String(); keypress == "ctrl+c" {
			m.Quitting = true
			return m, tea.Quit
		}

		return m, nil

	case tea.WindowSizeMsg:
		for _, p := range m.progressBars {
			p.progress.Width = msg.Width - padding*2 - 4
			if p.progress.Width > maxWidth {
				p.progress.Width = maxWidth
			}
		}
		return m, nil

	case UpdateProgressMsg:
		return m.processUpdateProgressMsg(msg)

	default:
		return m, nil
	}
}

func (m SizeModel) processUpdateProgressMsg(msg UpdateProgressMsg) (tea.Model, tea.Cmd) {
	ix := -1
	for k, p := range m.progressBars {
		if p.ID == msg.ID {
			ix = k
			break
		}
	}

	if ix == -1 {
		p := size{
			progress:       progress.New(progress.WithDefaultGradient()),
			SizeProperties: SizeProperties{ID: msg.ID}, //nolint:exhaustruct
		}
		m.progressBars = append(m.progressBars, p)
		ix = len(m.progressBars) - 1
	}

	p := m.progressBars[ix]
	p.Current = msg.Current
	p.Total = msg.Total
	p.Prefix = msg.Prefix
	p.Suffix = msg.Suffix
	m.progressBars[ix] = p

	return m, nil
}

// View renders the model.
func (m SizeModel) View() string {
	pad := strings.Repeat(" ", padding)
	var out []byte

	for _, p := range m.progressBars {
		out = append(out, pad+p.Prefix...)
		if p.Total > 0 {
			out = append(out, styles.ProgressTitleTextStyle.Render(
				p.progress.ViewAs(float64(p.Current)/float64(p.Total)),
			)...)
		}
		out = append(out, styles.ProgressTitleTextStyle.Render(p.Suffix)+"\n"...)
	}

	return string(out)
}

// NewSize creates new model.
func NewSize() SizeModel {
	m := SizeModel{
		progressBars: []size{},
		Quitting:     false,
	}
	return m
}
