// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func parseRawValue(rawValue bson.RawValue) string {
	if rawValue.IsZero() {
		return ""
	}
	var m any
	if err := bson.UnmarshalValue(rawValue.Type, rawValue.Value, &m); err == nil {
		if jsonValue, err := json.MarshalIndent(m, "", "    "); err == nil {
			return string(jsonValue)
		}
	}
	return rawValue.String()
}

func parseOption(commandRaw bson.Raw, key string) any {
	if opt := commandRaw.Lookup(key); !opt.IsZero() {
		var m any
		if err := bson.UnmarshalValue(opt.Type, opt.Value, &m); err == nil {
			return m
		}
	}
	return nil
}

func parseOptions(commandRaw bson.Raw, keys []string) string {
	opts := make(map[string]any)
	for _, key := range keys {
		if val := parseOption(commandRaw, key); val != nil {
			opts[key] = val
		}
	}

	if len(opts) > 0 {
		if optionsJSON, err := json.Marshal(opts); err == nil {
			return string(optionsJSON)
		}
	}
	return ""
}

func parseDocument(commandRaw bson.Raw, key string) string {
	if doc := commandRaw.Lookup(key); !doc.IsZero() {
		var m any
		if err := bson.UnmarshalValue(doc.Type, doc.Value, &m); err == nil {
			return parseRawValue(doc)
		}
	}
	return ""
}
