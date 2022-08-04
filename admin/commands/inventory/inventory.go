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

// Package inventory provides inventory commands.
package inventory

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

// Hide inventory commands from help by default.
const hide = true

// register commands
var (
	inventoryC       = kingpin.Command("inventory", "Inventory commands").Hide(hide)
	inventoryListC   = inventoryC.Command("list", "List inventory commands").Hide(hide)
	inventoryAddC    = inventoryC.Command("add", "Add to inventory commands").Hide(hide)
	inventoryRemoveC = inventoryC.Command("remove", "Remove from inventory commands").Hide(hide)
)

// formatTypeValue checks acceptable type value and variations contains input and returns type value.
// Values comparison is case-insensitive.
func formatTypeValue(acceptableTypeValues map[string][]string, input string) (*string, error) {
	if input == "" {
		return nil, nil
	}

	for value, variations := range acceptableTypeValues {
		variations = append(variations, value)
		for _, variation := range variations {
			if strings.EqualFold(variation, input) {
				return &value, nil
			}
		}
	}
	return nil, errors.Errorf("unexpected type value %q", input)
}
