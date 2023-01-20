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

package config

import (
	"sync"

	"github.com/sirupsen/logrus"
)

type storage struct {
	cfg *Config
	mu  sync.RWMutex
}

var cfgStorage = storage{}

// Get returns a global config object.
func Get() *Config {
	cfgStorage.mu.RLock()
	defer cfgStorage.mu.RUnlock()

	return cfgStorage.cfg
}

// Reload reloads config into the global object.
func Reload(l *logrus.Entry) (string, error) {
	cfgStorage.mu.Lock()
	defer cfgStorage.mu.Unlock()

	newCfg := &Config{}
	cfgPath, err := getFromCmdLine(newCfg, l)
	if err != nil {
		cfgStorage.cfg = newCfg
	}

	return cfgPath, err
}
