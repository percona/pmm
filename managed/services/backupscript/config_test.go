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

package backupscript

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// slaveEraKeys are the stale keys SEP's generator emits. If any appears in a
// rendered config, the payload would default XTRABACKUP_REPLICA_INFO to True and
// add --slave-info — the exact failure hit on the 8.4 node. This test is the
// regression guard the design calls for.
var slaveEraKeys = []string{
	"XTRABACKUP_SLAVE_INFO",
	"ONLY_IF_RUNNING_SLAVE",
	"XTRABACKUP_STOP_SLAVE",
}

func TestRenderConfig_UsesReplicaEraSchema(t *testing.T) {
	t.Parallel()

	out, err := RenderConfig(ConfigParams{
		Alias:                "mysql-svc",
		BackupDir:            "/root/backups",
		Compress:             true,
		CompressionAlgorithm: "zstd",
		Copies:               3,
		ReplicaInfo:          false,
		XtrabackupBinary:     "xtrabackup",
	})
	require.NoError(t, err)

	// Regression guard: no SLAVE-era key may ever be emitted.
	for _, k := range slaveEraKeys {
		assert.NotContainsf(t, out, k, "rendered config must not contain stale SLAVE-era key %s", k)
	}

	// The current REPLICA-era key must be present and set to the requested value.
	assert.Contains(t, out, "XTRABACKUP_REPLICA_INFO: false")
	assert.Contains(t, out, "BACKUP_DIR: /root/backups")
	assert.Contains(t, out, "XTRABACKUP_COPIES: 3")
	assert.Contains(t, out, "XTRABACKUP_BIN_CMD: xtrabackup")
	assert.Contains(t, out, "COMPRESSION_ALGORITHM: zstd")
}

func TestRenderConfig_ParsesWithPayloadSchema(t *testing.T) {
	t.Parallel()

	out, err := RenderConfig(ConfigParams{Alias: "svc", BackupDir: "/data"})
	require.NoError(t, err)

	// Mirror the payload's LocalConfig contract: ALL_SERVERS + a SERVER_LIST with
	// exactly one entry (so a --server filter flag is never needed).
	var doc struct {
		AllServers map[string]any   `yaml:"ALL_SERVERS"`
		ServerList []map[string]any `yaml:"SERVER_LIST"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(out), &doc))

	require.NotEmpty(t, doc.AllServers)
	require.Len(t, doc.ServerList, 1, "must emit exactly one server per job")
	assert.Equal(t, "svc", doc.ServerList[0]["ALIAS"])
	assert.Equal(t, "/data", doc.AllServers["BACKUP_DIR"])
	assert.Equal(t, defaultMyCnfPath, doc.ServerList[0]["DEFAULTS_FILE"])
	// Defaults applied.
	assert.Equal(t, "localhost", doc.ServerList[0]["HOST"])
	assert.Equal(t, 3306, doc.ServerList[0]["PORT"])
}

func TestRenderConfig_Defaults(t *testing.T) {
	t.Parallel()

	out, err := RenderConfig(ConfigParams{Alias: "svc", BackupDir: "/data"})
	require.NoError(t, err)
	assert.Contains(t, out, "COMPRESSION_ALGORITHM: "+defaultCompressionAlgorithm)
	assert.Contains(t, out, "XTRABACKUP_COPIES: 2")
	assert.Contains(t, out, "XTRABACKUP_BIN_CMD: xtrabackup")
}

func TestRenderConfig_Validation(t *testing.T) {
	t.Parallel()

	_, err := RenderConfig(ConfigParams{BackupDir: "/data"})
	assert.Error(t, err, "alias is required")

	_, err = RenderConfig(ConfigParams{Alias: "svc"})
	assert.Error(t, err, "backup_dir is required")
}

func TestBuildBackupJob_ShapeAndNoServerFlag(t *testing.T) {
	t.Parallel()

	yamlCfg, err := RenderConfig(ConfigParams{Alias: "svc", BackupDir: "/data"})
	require.NoError(t, err)

	job := buildBackupJob(buildJobParams{
		RunID:         "run-123",
		ServiceID:     "/service_id/abc",
		ConfigVersion: 4,
		NodeName:      "db-node-1",
		RenderedYAML:  yamlCfg,
	})

	assert.Equal(t, "batch", job.Type)
	assert.Equal(t, "pmm-backup-run-123", job.ID)
	// Service IDs contain slashes; they must not leak into the job id.
	assert.NotContains(t, job.ID, "/")
	assert.Equal(t, "run-123", job.Meta["backup_run_id"])

	require.Len(t, job.Constraints, 1)
	assert.Equal(t, "${node.unique.name}", job.Constraints[0].LTarget)
	assert.Equal(t, "db-node-1", job.Constraints[0].RTarget)

	require.Len(t, job.TaskGroups, 1)
	tg := job.TaskGroups[0]
	assert.Equal(t, 0, tg.RestartPolicy.Attempts, "a failed backup must never silently retry")
	assert.Equal(t, 0, tg.ReschedulePolicy.Attempts)

	require.Len(t, tg.Tasks, 1)
	task := tg.Tasks[0]
	assert.Equal(t, "raw_exec", task.Driver)

	// Config is delivered via an embedded template, not an artifact stanza.
	require.Len(t, task.Templates, 1)
	assert.Equal(t, "local/backup_config.yaml", task.Templates[0].DestPath)
	assert.Equal(t, yamlCfg, task.Templates[0].EmbeddedTmpl)

	// Regression guard: the invocation must never carry a --server flag (the
	// current payload has no such flag; SEP's stored invocation would error).
	args, ok := task.Config["args"].([]string)
	require.True(t, ok)
	for _, a := range args {
		assert.NotEqual(t, "--server", a, "current payload has no --server flag")
		assert.NotContains(t, a, "--server=")
	}
	assert.Contains(t, strings.Join(args, " "), "-c ${NOMAD_TASK_DIR}/backup_config.yaml")
}

func TestParseManifest(t *testing.T) {
	t.Parallel()

	stdout := "starting backup\n" +
		manifestMarker + `{"run_id":"r1","status":"success","backup_dir":"/root/backups/x","size_bytes":12345}` + "\n" +
		"done\n"
	m := parseManifest(stdout)
	require.NotNil(t, m)
	assert.Equal(t, "success", m.Status)
	assert.Equal(t, "/root/backups/x", m.BackupDir)
	assert.Equal(t, int64(12345), m.SizeBytes)

	assert.Nil(t, parseManifest("no manifest here"))
}
