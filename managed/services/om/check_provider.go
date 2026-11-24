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

package om

import (
	"context"
	"time"

	"github.com/percona/pmm/managed/models"
)

// CheckSubject names one host a kind's readiness facts are read for, as a fact subject kind and key.
type CheckSubject struct {
	Kind string
	Key  string
}

// factObservationReader reads a subject's current facts under a profile; E0-T4's factReader fulfils it.
type factObservationReader interface {
	Observation(ctx context.Context, subjectKind, subjectKey, profile string) (map[string]any, time.Time, error)
}

// factCheckProvider fulfils CheckProvider: it returns a kind's declared check set and reads the kind's
// subjects' facts through E0-T4's reader, aggregated across the subjects.
type factCheckProvider struct {
	reader factObservationReader
	now    func() time.Time
}

// newFactCheckProvider builds a check provider that reads facts through the given reader.
func newFactCheckProvider(reader factObservationReader) *factCheckProvider {
	return &factCheckProvider{reader: reader, now: time.Now}
}

// CheckInputs returns the kind's check set and its declared facts aggregated across the subjects: a
// boolean passes only when every subject does, a number is the least, a fact missing anywhere is dropped.
func (p *factCheckProvider) CheckInputs(ctx context.Context, op *models.OmOperation, kind OperationKind) (CheckSet, map[string]any, map[string]time.Duration, error) {
	profile := op.Kind + "_check"
	subjects := kind.Subjects(op)
	perSubject := make([]map[string]any, 0, len(subjects))
	subjectAges := make([]time.Duration, 0, len(subjects))
	now := p.now()
	for _, subject := range subjects {
		facts, collectedAt, err := p.reader.Observation(ctx, subject.Kind, subject.Key, profile)
		if err != nil {
			return CheckSet{}, nil, nil, err
		}
		perSubject = append(perSubject, facts)
		subjectAges = append(subjectAges, now.Sub(collectedAt))
	}
	facts := make(map[string]any, len(kind.CheckSet.Facts))
	ages := make(map[string]time.Duration, len(kind.CheckSet.Facts))
	for _, declared := range kind.CheckSet.Facts {
		value, age, ok := aggregateFact(declared.Name, perSubject, subjectAges)
		if !ok {
			continue
		}
		facts[declared.Name] = value
		ages[declared.Name] = age
	}
	return kind.CheckSet, facts, ages, nil
}

// aggregateFact folds one fact across the subjects (booleans by AND, numbers by the least, age by the
// oldest), reporting not-ok when a subject lacks it or the types disagree so it reads as missing.
func aggregateFact(name string, perSubject []map[string]any, ages []time.Duration) (any, time.Duration, bool) {
	if len(perSubject) == 0 {
		return nil, 0, false
	}
	var oldest time.Duration
	allBool, andValue := true, true
	allNumber, leastNumber, haveNumber := true, 0.0, false
	for i, facts := range perSubject {
		value, present := facts[name]
		if !present {
			return nil, 0, false
		}
		if ages[i] > oldest {
			oldest = ages[i]
		}
		if boolean, ok := value.(bool); ok {
			andValue = andValue && boolean
		} else {
			allBool = false
		}
		if number, ok := value.(float64); ok {
			if !haveNumber || number < leastNumber {
				leastNumber, haveNumber = number, true
			}
		} else {
			allNumber = false
		}
	}
	if allBool {
		return andValue, oldest, true
	}
	if allNumber {
		return leastNumber, oldest, true
	}
	return nil, 0, false
}
