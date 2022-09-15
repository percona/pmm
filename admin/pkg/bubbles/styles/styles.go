// Copyright 2019 Percona LLC
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

// Package styles holds common styles for BubbleTea programs
package styles

import "github.com/charmbracelet/lipgloss"

var (
	ParagraphTextStyle         = lipgloss.NewStyle().Margin(1, 0, 1, 4)
	ParagraphNoMarginTextStyle = lipgloss.NewStyle().Margin(0, 0, 0, 4)
	ProgressTitleTextStyle     = lipgloss.NewStyle().Margin(0, 0, 0, 4)
	QuitTextStyle              = lipgloss.NewStyle().Margin(1, 0, 2, 4)
	SuccessTextStyle           = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	SuccessBox = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(1, 2).
			Margin(1, 0, 2, 4)
)
