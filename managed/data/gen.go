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

// Package data provides access to embedded data.
package data

import "embed"

// IATemplates holds IA template files in the struct of type embed.FS which implements the io/fs package's FS interface.
//
//go:embed iatemplates/*
var IATemplates embed.FS

// OLMCRDs ...
//
//go:embed crds/*
var OLMCRDs embed.FS
