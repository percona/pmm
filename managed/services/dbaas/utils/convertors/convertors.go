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

// Package convertors provides data size convert functinality.
package convertors

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

const (
	kiloByte uint64 = 1000
	kibiByte uint64 = 1024
	megaByte uint64 = kiloByte * 1000
	mibiByte uint64 = kibiByte * 1024
	gigaByte uint64 = megaByte * 1000
	gibiByte uint64 = mibiByte * 1024
	teraByte uint64 = gigaByte * 1000
	tebiByte uint64 = gibiByte * 1024
	petaByte uint64 = teraByte * 1000
	pebiByte uint64 = tebiByte * 1024
	exaByte  uint64 = petaByte * 1000
	exbiByte uint64 = pebiByte * 1024
)

// StrToBytes converts string containing memory as string to number of bytes the string represents.
func StrToBytes(memory string) (uint64, error) {
	if len(memory) == 0 {
		return 0, nil
	}
	i := len(memory) - 1
	for i >= 0 && !unicode.IsDigit(rune(memory[i])) {
		i--
	}
	var suffix string
	if i >= 0 {
		suffix = memory[i+1:]
	}

	// Resources units map for k8s
	//
	// Supports the following units
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#meaning-of-memory
	//
	// Support of 'm' unit can be redundant because it's used for CPU limits mostly
	suffixMapping := map[string]float64{
		"m":  0.001,
		"k":  float64(kiloByte),
		"Ki": float64(kibiByte),
		"M":  float64(megaByte),
		"Mi": float64(mibiByte),
		"G":  float64(gigaByte),
		"Gi": float64(gibiByte),
		"T":  float64(teraByte),
		"Ti": float64(tebiByte),
		"P":  float64(petaByte),
		"Pi": float64(pebiByte),
		"E":  float64(exaByte),
		"Ei": float64(exbiByte),
		"":   1.0,
	}
	coeficient, ok := suffixMapping[suffix]
	if !ok {
		return 0, errors.Errorf("suffix '%s' is not supported", suffix)
	}

	if suffix != "" {
		memory = memory[:i+1]
	}
	value, err := strconv.ParseFloat(memory, 64)
	if err != nil {
		return 0, errors.Errorf("given value '%s' is not a number", memory)
	}
	return uint64(math.Ceil(value * coeficient)), nil
}

// StrToMilliCPU converts CPU as a string representation to millicpus represented as an integer.
func StrToMilliCPU(cpu string) (uint64, error) {
	if cpu == "" {
		return 0, nil
	}
	if strings.HasSuffix(cpu, "m") {
		cpu = cpu[:len(cpu)-1]
		millis, err := strconv.ParseUint(cpu, 10, 64)
		if err != nil {
			return 0, err
		}
		return millis, nil
	}
	floatCPU, err := strconv.ParseFloat(cpu, 64)
	if err != nil {
		return 0, err
	}
	return uint64(floatCPU * 1000), nil
}

// BytesToStr converts integer of bytes to string.
func BytesToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

// MilliCPUToStr converts integer of milli CPU to string.
func MilliCPUToStr(i int32) string {
	return strconv.FormatInt(int64(i), 10) + "m"
}
