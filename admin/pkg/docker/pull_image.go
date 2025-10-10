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

package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"io"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/image"

	"github.com/percona/pmm/admin/pkg/bubbles/progress"
)

// PullImage pulls image from Docker registry.
func (b *Base) PullImage(ctx context.Context, dockerImage string, opts image.PullOptions) (io.Reader, error) {
	return b.Cli.ImagePull(ctx, dockerImage, opts)
}

// StatusMsg is a struct to unmarshal Docker json status to.
type StatusMsg struct {
	Status         string `json:"status"`
	ID             string `json:"id"`
	ProgressDetail *struct {
		Current *int `json:"current"`
		Total   *int `json:"total"`
	} `json:"progressDetail"`
}

// ParsePullImageProgress parses Docker json status from reader and sends
// progress messages to a BubbleTea program.
func (b *Base) ParsePullImageProgress(r io.Reader, p *tea.Program) (<-chan struct{}, <-chan error) {
	doneChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			var s *StatusMsg
			if err := json.Unmarshal(scanner.Bytes(), &s); err != nil {
				continue
			}

			if s.ProgressDetail == nil {
				continue
			}

			total := 0
			current := 0

			if s.ProgressDetail.Current != nil && s.ProgressDetail.Total != nil {
				total = *s.ProgressDetail.Total
				current = *s.ProgressDetail.Current
			}

			p.Send(progress.UpdateProgressMsg{
				SizeProperties: progress.SizeProperties{
					ID:      s.ID,
					Total:   total,
					Current: current,
					Prefix:  s.ID,
					Suffix:  s.Status,
				},
			})
		}

		doneChan <- struct{}{}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return doneChan, errChan
}
