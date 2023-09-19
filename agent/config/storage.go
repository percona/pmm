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

package config

import (
	"sync"

	"github.com/sirupsen/logrus"
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

// Get returns config.
func (s *Storage) Get() *Config {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.cfg
}

// Reload reloads config.
func (s *Storage) Reload(l *logrus.Entry) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	newCfg := &Config{}
	cfgPath, err := getFromCmdLine(newCfg, l)
	if err != nil {
		if _, ok := err.(ConfigFileDoesNotExistError); !ok { //nolint:errorlint
			return cfgPath, err
		}
	}

	s.cfg = newCfg

	return cfgPath, err
}
