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
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
)

func TestParseArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		validate func(t *testing.T, result string)
	}{
		{
			name: "valid array with documents",
			json: `{"arr": [{"name": "test", "value": 42}, {"name": "test2", "value": 100}]}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "name")
				require.Contains(t, result, "test")
				require.Contains(t, result, "value")
				require.Contains(t, result, "42")
			},
		},
		{
			name: "array with single document",
			json: `{"arr": [{"field": "value"}]}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "field")
				require.Contains(t, result, "value")
			},
		},
		{
			name: "empty array",
			json: `{"arr": []}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				// Empty array falls through to doc.String() which returns "[]"
				require.Equal(t, "[]", result)
			},
		},
		{
			name: "array with non-document values (strings, numbers)",
			json: `{"arr": ["string", 123, true]}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				// Non-document values won't unmarshal to maps, so we get doc.String()
				require.NotEmpty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Parse the JSON to BSON
			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			// Get the array field
			arrayVal := raw.Lookup("arr")
			require.False(t, arrayVal.IsZero(), "Expected 'arr' field to exist")

			// Test parseArray
			result := parseArray(arrayVal)
			tt.validate(t, result)
		})
	}
}

func TestParseArray_InvalidInput(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		validate func(t *testing.T, result string)
	}{
		{
			name: "not an array (string value)",
			json: `{"field": "not an array"}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				// Should return doc.String() since it's not an array
				require.NotEmpty(t, result)
			},
		},
		{
			name: "not an array (document value)",
			json: `{"field": {"nested": "doc"}}`,
			validate: func(t *testing.T, result string) {
				t.Helper()
				// Should return doc.String() since it's not an array
				require.NotEmpty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			fieldVal := raw.Lookup("field")
			require.False(t, fieldVal.IsZero())

			result := parseArray(fieldVal)
			tt.validate(t, result)
		})
	}
}

func TestParseRawValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		field    string
		validate func(t *testing.T, result string)
	}{
		{
			name:  "zero value",
			json:  `{}`,
			field: "missing",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.Empty(t, result)
			},
		},
		{
			name:  "string value",
			json:  `{"value": "test string"}`,
			field: "value",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "test string")
			},
		},
		{
			name:  "number value",
			json:  `{"number": 42}`,
			field: "number",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "42")
			},
		},
		{
			name:  "boolean value",
			json:  `{"flag": true}`,
			field: "flag",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			val := raw.Lookup(tt.field)
			result := parseRawValue(val)
			tt.validate(t, result)
		})
	}
}

func TestParseOption(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		key      string
		validate func(t *testing.T, result any)
	}{
		{
			name: "existing key",
			json: `{"ordered": true, "maxTimeMS": 1000}`,
			key:  "ordered",
			validate: func(t *testing.T, result any) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, true, result)
			},
		},
		{
			name: "missing key",
			json: `{"ordered": true}`,
			key:  "missing",
			validate: func(t *testing.T, result any) {
				t.Helper()
				require.Nil(t, result)
			},
		},
		{
			name: "numeric key",
			json: `{"maxTimeMS": 5000}`,
			key:  "maxTimeMS",
			validate: func(t *testing.T, result any) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, int32(5000), result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			result := parseOption(raw, tt.key)
			tt.validate(t, result)
		})
	}
}

func TestParseOptions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		keys     []string
		validate func(t *testing.T, result string)
	}{
		{
			name: "multiple keys exist",
			json: `{"ordered": true, "maxTimeMS": 1000}`,
			keys: []string{"ordered", "maxTimeMS"},
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "ordered")
				require.Contains(t, result, "maxTimeMS")
			},
		},
		{
			name: "some keys missing",
			json: `{"ordered": true}`,
			keys: []string{"ordered", "missing"},
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "ordered")
				require.NotContains(t, result, "missing")
			},
		},
		{
			name: "no keys exist",
			json: `{"other": "value"}`,
			keys: []string{"missing1", "missing2"},
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.Empty(t, result)
			},
		},
		{
			name: "empty keys list",
			json: `{"field": "value"}`,
			keys: []string{},
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			result := parseOptions(raw, tt.keys)
			tt.validate(t, result)
		})
	}
}

func TestParseEmbeddedDocument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		field    string
		validate func(t *testing.T, result string)
	}{
		{
			name:  "simple document",
			json:  `{"doc": {"name": "test", "value": 42}}`,
			field: "doc",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "name")
				require.Contains(t, result, "test")
				require.Contains(t, result, "value")
				require.Contains(t, result, "42")
			},
		},
		{
			name:  "nested document",
			json:  `{"doc": {"outer": {"inner": "value"}}}`,
			field: "doc",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "outer")
				require.Contains(t, result, "inner")
			},
		},
		{
			name:  "empty document",
			json:  `{"doc": {}}`,
			field: "doc",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			doc := raw.Lookup(tt.field)
			require.False(t, doc.IsZero())

			result := parseEmbeddedDocument(doc)
			tt.validate(t, result)
		})
	}
}

func TestParseDocument(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		json     string
		key      string
		validate func(t *testing.T, result string)
	}{
		{
			name: "document field",
			json: `{"doc": {"name": "test", "value": 42}}`,
			key:  "doc",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "name")
				require.Contains(t, result, "test")
			},
		},
		{
			name: "array field",
			json: `{"arr": [{"item": 1}, {"item": 2}]}`,
			key:  "arr",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "item")
			},
		},
		{
			name: "string field",
			json: `{"text": "value"}`,
			key:  "text",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.NotEmpty(t, result)
				require.Contains(t, result, "value")
			},
		},
		{
			name: "missing field",
			json: `{"other": "value"}`,
			key:  "missing",
			validate: func(t *testing.T, result string) {
				t.Helper()
				require.Empty(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			vr, err := bsonrw.NewExtJSONValueReader(bytes.NewReader([]byte(tt.json)), true)
			require.NoError(t, err)

			dec, err := bson.NewDecoder(vr)
			require.NoError(t, err)

			var raw bson.Raw
			err = dec.Decode(&raw)
			require.NoError(t, err)

			result := parseDocument(raw, tt.key)
			tt.validate(t, result)
		})
	}
}
