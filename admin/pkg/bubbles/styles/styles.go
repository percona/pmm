// Copyright (C) 2024 Percona LLC
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

// Package styles holds common styles for BubbleTea programs.
package styles

import "github.com/charmbracelet/lipgloss"

// ProgressTitleTextStyle styles progress bar's title.
var ProgressTitleTextStyle = lipgloss.NewStyle().Margin(0, 0, 0, 4)
