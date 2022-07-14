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

package client

import (
	"context"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"

	"github.com/percona/pmm/agent/utils/templates"
	"github.com/percona/pmm/api/agentpb"
)

const pbmBin = "pbm"

func (c *Client) handlePBMSwitchRequest(ctx context.Context, req *agentpb.PBMSwitchPITRRequest, id uint32) error {
	c.l.Infof("Switching pbm Point-in-Time Recovery feature to the state enabled: %t", req.Enabled)
	if _, err := exec.LookPath(pbmBin); err != nil {
		return errors.Wrapf(err, "lookpath: %s", pbmBin)
	}

	dsn, err := templates.RenderDSN(req.Dsn, req.TextFiles, filepath.Join(c.cfg.Paths.TempDir, "pbm-switch-pitr", strconv.Itoa(int(id))))
	if err != nil {
		return errors.WithStack(err)
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	output, err := exec.CommandContext(
		ctx,
		pbmBin,
		"config",
		"--set",
		"pitr.enabled="+strconv.FormatBool(req.Enabled),
		"--mongodb-uri="+dsn).
		CombinedOutput() // #nosec G204
	if err != nil {
		return errors.Wrapf(err, "pbm config error: %s", string(output))
	}

	return nil
}
