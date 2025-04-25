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

// Package fingerprinter is used to get fingerpint for queries.
package fingerprinter

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/percona/percona-toolkit/src/go/mongolib/fingerprinter"
	"github.com/percona/percona-toolkit/src/go/mongolib/proto"
	"go.mongodb.org/mongo-driver/bson"
)

// ProfilerFingerprinter holds any necessary configuration or dependencies.
type ProfilerFingerprinter struct {
	keyFilters []string
	// Add fields here if you need to configure the fingerprinter
}

// NewFingerprinter creates a new instance of ProfilerFingerprinter.
func NewFingerprinter(keyFilters []string) *ProfilerFingerprinter {
	return &ProfilerFingerprinter{
		keyFilters: keyFilters,
	}
}

// Fingerprint generates a unique MongoDB command fingerprint from profiler output.
func (pf *ProfilerFingerprinter) Fingerprint(doc proto.SystemProfile) (fingerprinter.Fingerprint, error) {
	fp := fingerprinter.Fingerprint{
		Namespace: doc.Ns,
		Operation: doc.Op,
	}

	// Parse the namespace to separate database and collection names
	parts := strings.SplitN(doc.Ns, ".", 2)
	fp.Database = parts[0]
	if len(parts) > 1 {
		fp.Collection = parts[1]
	}

	// Select operation type and build command with optional fields
	switch doc.Op {
	case "query":
		return pf.fingerprintFind(fp, doc)
	case "insert":
		return pf.fingerprintInsert(fp)
	case "update":
		return pf.fingerprintUpdate(fp, doc)
	case "delete", "remove":
		return pf.fingerprintDelete(fp, doc)
	case "command":
		return pf.fingerprintCommand(fp, doc)
	default:
		return pf.fingerprintCommand(fp, doc)
	}
}

// Helper for find operations with optional parameters.
func (pf *ProfilerFingerprinter) fingerprintFind(fp fingerprinter.Fingerprint, doc proto.SystemProfile) (fingerprinter.Fingerprint, error) {
	filter := ""
	command := doc.Command.Map()
	if f, ok := command["filter"]; ok {
		values := maskValues(f, make(map[string]maskOption))
		filterJSON, _ := json.Marshal(values)
		filter = string(filterJSON)
	}

	// Initialize mongosh command with required fields
	fp.Fingerprint = fmt.Sprintf(`db.%s.find(%s`, fp.Collection, filter)
	fp.Keys = filter

	// Optional fields for find command
	if command["project"] != nil {
		projectionJSON, _ := json.Marshal(command["project"])
		fp.Fingerprint += fmt.Sprintf(`, %s`, projectionJSON)
	}
	fp.Fingerprint += ")"

	if sort, ok := command["sort"]; ok {
		sortJSON, _ := json.Marshal(sort.(bson.D).Map()) // TODO deprecated in GO Driver 2.0
		fp.Fingerprint += fmt.Sprintf(`.sort(%s)`, sortJSON)
	}
	if _, ok := command["limit"]; ok {
		fp.Fingerprint += `.limit(?)`
	}
	if _, ok := command["skip"]; ok {
		fp.Fingerprint += `.skip(?)`
	}
	if batchSize, ok := command["batchSize"]; ok {
		fp.Fingerprint += fmt.Sprintf(`.batchSize(%d)`, batchSize)
	}

	return fp, nil
}

// Helper for insert operations
func (pf *ProfilerFingerprinter) fingerprintInsert(fp fingerprinter.Fingerprint) (fingerprinter.Fingerprint, error) {
	fp.Fingerprint = fmt.Sprintf(`db.%s.insert(?)`, fp.Collection)
	return fp, nil
}

// Helper for update operations
func (pf *ProfilerFingerprinter) fingerprintUpdate(fp fingerprinter.Fingerprint, doc proto.SystemProfile) (fingerprinter.Fingerprint, error) {
	command := doc.Command.Map()
	filterJSON, _ := json.Marshal(maskValues(command["q"], make(map[string]maskOption)))
	updateJSON, _ := json.Marshal(maskValues(command["u"], make(map[string]maskOption)))

	fp.Fingerprint = fmt.Sprintf(`db.%s.update(%s, %s`, fp.Collection, filterJSON, updateJSON)
	fp.Keys = string(filterJSON)

	if command["upsert"] == true || command["multi"] == true {
		options := make(map[string]interface{})
		if command["upsert"] == true {
			options["upsert"] = true
		}
		if command["multi"] == true {
			options["multi"] = true
		}
		optionsJSON, _ := json.Marshal(options)
		fp.Fingerprint += fmt.Sprintf(`, %s`, optionsJSON)
	}
	fp.Fingerprint += ")"

	return fp, nil
}

// Helper for delete operations
func (pf *ProfilerFingerprinter) fingerprintDelete(fp fingerprinter.Fingerprint, doc proto.SystemProfile) (fingerprinter.Fingerprint, error) {
	command := doc.Command.Map()
	method := "deleteMany"
	if limit, ok := command["limit"]; ok && limit == int32(1) {
		method = "deleteOne"
	}
	filterJSON, _ := json.Marshal(maskValues(command["q"], make(map[string]maskOption)))
	fp.Fingerprint = fmt.Sprintf(`db.%s.%s(%s)`, fp.Collection, method, filterJSON)
	fp.Keys = string(filterJSON)
	return fp, nil
}

// Helper for general command operations, including support for "aggregate" commands
func (pf *ProfilerFingerprinter) fingerprintCommand(fp fingerprinter.Fingerprint, doc proto.SystemProfile) (fingerprinter.Fingerprint, error) {
	// Unmarshal the command into a map for easy access and manipulation
	command := doc.Command.Map()

	maskOptions := map[string]maskOption{
		"$db":                      {remove: true},
		"$readPreference":          {remove: true},
		"$readConcern":             {remove: true},
		"$writeConcern":            {remove: true},
		"$clusterTime":             {remove: true},
		"$oplogQueryData":          {remove: true},
		"$replData":                {remove: true},
		"lastKnownCommittedOpTime": {remove: true},
		"lsid":                     {remove: true},
		"findAndModify":            {skipMask: true},
		"remove":                   {skipMask: true},
	}
	if _, exists := command["aggregate"]; exists {
		// Set collection and initialize aggregation structure
		fp.Fingerprint = fmt.Sprintf(`db.%s.aggregate([`, fp.Collection)
		stageStrings := []string{}

		// Process pipeline stages, replacing all values with "?"
		if pipeline, exists := command["pipeline"]; exists {
			pipelineStages, _ := pipeline.(bson.A)

			for _, stage := range pipelineStages {
				stageMap := stage.(bson.D).Map()
				var stageJSON []byte
				switch {
				case stageMap["$match"] != nil:
					stageJSON, _ = json.Marshal(maskValues(stageMap, maskOptions))
				default:
					stageJSON, _ = bson.MarshalExtJSON(stageMap, false, false)
				}

				stageStrings = append(stageStrings, string(stageJSON))
			}

			fp.Fingerprint += strings.Join(stageStrings, ", ")
		}
		fp.Fingerprint += "])"
		if collation, exists := command["collation"]; exists {
			collationMasked, _ := json.Marshal(maskValues(collation, maskOptions))
			fp.Fingerprint += fmt.Sprintf(`, collation: %s`, collationMasked)
		}

		// Build a descriptive Keys field
		fp.Keys = strings.Join(stageStrings, ", ")
	} else {
		// Handle other commands generically
		commandMasked, _ := json.Marshal(maskValues(doc.Command, maskOptions))
		fp.Fingerprint = fmt.Sprintf(`db.runCommand(%s)`, commandMasked)
		fp.Keys = string(commandMasked)
	}

	return fp, nil
}

type maskOption struct {
	remove   bool
	skipMask bool
}

// maskValues replaces all values within a map or slice with "?" recursively and removes keys in the filter.
func maskValues(data interface{}, options map[string]maskOption) interface{} {
	switch v := data.(type) {
	case bson.D:
		masked := make(bson.M)
		for _, value := range v {
			option, ok := options[value.Key]
			switch {
			case ok && option.remove:
				continue
			case ok && option.skipMask:
				masked[value.Key] = value.Value
			default:
				masked[value.Key] = maskValues(value.Value, options)
			}
		}
		return masked
	case bson.M:
		masked := make(bson.M)
		for key, value := range v {
			option, ok := options[key]
			switch {
			case ok && option.remove:
				continue
			case ok && option.skipMask:
				masked[key] = value
			default:
				masked[key] = maskValues(value, options)
			}
		}
		return masked
	case bson.A:
		for i := range v {
			v[i] = maskValues(v[i], options)
		}
		return v
	default:
		return "?"
	}
}

// DefaultKeyFilters returns default keys used for filtering.
func DefaultKeyFilters() []string {
	return []string{}
}
