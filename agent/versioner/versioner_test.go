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

package versioner

import (
	"os/exec"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockedExec struct {
	Output []byte
}

func (m *mockedExec) CombinedOutput() ([]byte, error) {
	return m.Output, nil
}

func TestVersioner(t *testing.T) {
	execMock := &MockExecFunctions{}
	versioner := New(execMock)

	t.Run("not found", func(t *testing.T) {
		execMock.On("LookPath", mysqldBin).Return("", &exec.Error{Err: exec.ErrNotFound}).Once()

		version, err := versioner.MySQLdVersion()
		assert.True(t, errors.Is(err, ErrNotFound))
		assert.Equal(t, "", version)
	})
	t.Run("mysqld", func(t *testing.T) {
		mysqldVersionOutput := []byte(`/usr/sbin/mysqld  Ver 8.0.22-13 for Linux on x86_64 (Percona Server (GPL), Release '13', Revision '6f7822f')
`)
		execMock.On("LookPath", mysqldBin).Return("", nil).Once()
		execMock.On("CommandContext", mock.Anything, mysqldBin, "--version").
			Return(&mockedExec{Output: mysqldVersionOutput}).Once()
		version, err := versioner.MySQLdVersion()
		assert.NoError(t, err)
		assert.Equal(t, "8.0.22-13", version)
	})
	t.Run("xtrabackup 2", func(t *testing.T) {
		mysqldVersionOutput := []byte(`xtrabackup: recognized server arguments: --datadir=/var/lib/mysql
xtrabackup version 2.4.23 based on MySQL server 5.7.34 Linux (x86_64) (revision id: 3320f39)
`)
		execMock.On("LookPath", xtrabackupBin).Return("", nil).Once()
		execMock.On("CommandContext", mock.Anything, xtrabackupBin, "--version").
			Return(&mockedExec{Output: mysqldVersionOutput}).Once()
		version, err := versioner.XtrabackupVersion()
		assert.NoError(t, err)
		assert.Equal(t, "2.4.23", version)
	})
	t.Run("xtrabackup 8", func(t *testing.T) {
		mysqldVersionOutput := []byte(`xtrabackup version 8.0.23-16 based on MySQL server 8.0.23 Linux (x86_64) (revision id: 934bc8f)
`)
		execMock.On("LookPath", xtrabackupBin).Return("", nil).Once()
		execMock.On("CommandContext", mock.Anything, xtrabackupBin, "--version").
			Return(&mockedExec{Output: mysqldVersionOutput}).Once()
		version, err := versioner.XtrabackupVersion()
		assert.NoError(t, err)
		assert.Equal(t, "8.0.23-16", version)
	})
	t.Run("xbcloud", func(t *testing.T) {
		mysqldVersionOutput := []byte(`xbcloud  Ver 8.0.23-16 for Linux (x86_64) (revision id: 934bc8f)
`)
		execMock.On("LookPath", xbcloudBin).Return("", nil).Once()
		execMock.On("CommandContext", mock.Anything, xbcloudBin, "--version").
			Return(&mockedExec{Output: mysqldVersionOutput}).Once()
		version, err := versioner.XbcloudVersion()
		assert.NoError(t, err)
		assert.Equal(t, "8.0.23-16", version)
	})
	t.Run("qpress", func(t *testing.T) {
		mysqldVersionOutput := []byte(`qpress 1.1 - Copyright 2006-2010 Lasse Reinhold - www.quicklz.com
Using QuickLZ 1.4.1 compression library
Compiled for: Windows [*nix]    [x86/x64] RISC    32-bit [64-bit]
...
`)
		execMock.On("LookPath", qpressBin).Return("", nil).Once()
		execMock.On("CommandContext", mock.Anything, qpressBin).
			Return(&mockedExec{Output: mysqldVersionOutput}).Once()
		version, err := versioner.Qpress()
		assert.NoError(t, err)
		assert.Equal(t, "1.1", version)
	})

	mock.AssertExpectationsForObjects(t, execMock)
}
