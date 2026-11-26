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

// Package poll provides helpers for polling until a condition is met.
package poll

import (
	"context"
	"fmt"
	"time"
)

// ConditionFunc returns true when the condition is successfully met.
type ConditionFunc func(ctx context.Context) (done bool, err error)

// UntilContextTimeout polls until condition returns done=true, err!=nil, or ctx is canceled.
func UntilContextTimeout(ctx context.Context, interval time.Duration, condition ConditionFunc) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be positive: %s", interval)
	}

	err := ctx.Err()
	if err != nil {
		return err
	}

	done, err := condition(ctx)
	if err != nil {
		return err
	}
	if done {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err = ctx.Err()
			if err != nil {
				return err
			}

			done, err := condition(ctx)
			if err != nil {
				return err
			}
			if done {
				return nil
			}
		}
	}
}
