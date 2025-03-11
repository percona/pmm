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

// Package errors contains common errors definitions.
package errors

import "github.com/pkg/errors"

var (
	// ErrInvalidArgument is returned when an invalid or unknown argument is specified.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrActionQueueOverflow is returned when the agent is already running the maximum number of actions.
	ErrActionQueueOverflow = errors.New("action queue overflow")

	// ErrActionUnimplemented is returned when action type is not handled/implemented.
	ErrActionUnimplemented = errors.New("action is not implemented")
)
