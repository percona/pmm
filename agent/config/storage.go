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

// Getter allows for getting a config.
type Getter interface {
	Get() *Config
}

// GetReloader allows for getting and reloading a config.
type GetReloader interface {
	Get() *Config
	Reload(l *logrus.Entry) (string, error)
}

// Check interfaces.
var (
	_ Getter      = &Storage{}
	_ GetReloader = &Storage{}
)

// NewStorage creates a new instance of Storage with optional initial config.
func NewStorage(cfg *Config) *Storage {
	return &Storage{
		cfg: cfg,
	}
}

// Storage holds config.
type Storage struct {
	cfg *Config
	mu  sync.RWMutex
}

// Get returns a global config object.
func (s *Storage) Get() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

// Reload reloads config into the global object.
func (s *Storage) Reload(l *logrus.Entry) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newCfg := &Config{}
	cfgPath, err := getFromCmdLine(newCfg, l)
	if err != nil {
		s.cfg = newCfg
	}

	return cfgPath, err
}
