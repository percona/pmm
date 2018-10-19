// pmm-managed
// Copyright (C) 2017 Percona LLC
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

package telemetry

import (
	"context"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if ok, _ := strconv.ParseBool(os.Getenv("INTEGRATION_TESTS")); !ok {
		t.Skipf("Environment variable INTEGRATION_TESTS is not set. Skipping integration test.")
	}

	if os.Getenv(envURL) == "" {
		t.Skipf("Environment variable %s is not set. Skipping integration test.", envURL)
	}

	uuid, err := GenerateUUID()
	require.NoError(t, err)
	s := NewService(uuid, "1.3.1")
	assert.True(t, s.init())
	assert.NoError(t, s.sendOnce(context.Background()))
}

func TestMakePayload(t *testing.T) {
	s := NewService("ECAB81E4C47D456CA9EC20AEBF91AB44", "1.3.1")
	expected := "ECAB81E4C47D456CA9EC20AEBF91AB44;OS;\nECAB81E4C47D456CA9EC20AEBF91AB44;PMM;1.3.1\n"
	assert.Equal(t, expected, string(s.makePayload())) // \n are important
}

func TestGetLinuxDistribution(t *testing.T) {
	for expected, procVersion := range map[string][]string{
		// cat /proc/version
		"Ubuntu 16.04": {
			`Linux version 4.4.0-47-generic (buildd@lcy01-03) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.2) ) #68-Ubuntu SMP Wed Oct 26 19:39:52 UTC 2016`,
			`Linux version 4.4.0-57-generic (buildd@lgw01-54) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #78-Ubuntu SMP Fri Dec 9 23:50:32 UTC 2016`,
			`Linux version 4.4.0-96-generic (buildd@lgw01-10) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #119-Ubuntu SMP Tue Sep 12 14:59:54 UTC 2017`,
			`Linux version 4.10.0-27-generic (buildd@lgw01-60) (gcc version 5.4.0 20160609 (Ubuntu 5.4.0-6ubuntu1~16.04.4) ) #30~16.04.2-Ubuntu SMP Thu Jun 29 16:07:46 UTC 2017`,
		},

		"Ubuntu 14.04": {
			`Linux version 3.13.0-129-generic (buildd@lgw01-05) (gcc version 4.8.4 (Ubuntu 4.8.4-2ubuntu1~14.04.3) ) #178-Ubuntu SMP Fri Aug 11 12:48:20 UTC 2017`,
		},

		"Ubuntu": {
			/* 18.10 beta */ `Linux version 4.13.0-11-lowlatency (buildd@lgw01-23) (gcc version 7.2.0 (Ubuntu 7.2.0-3ubuntu1)) #12-Ubuntu SMP PREEMPT Tue Sep 12 17:37:56 UTC 2017`,
			/* 14.04.5    */ `Linux version 3.13.0-77-generic (buildd@lcy01-30) (gcc version 4.8.2 (Ubuntu 4.8.2-19ubuntu1) ) #121-Ubuntu SMP Wed Jan 20 10:50:42 UTC 2016`,
		},

		"Debian": {
			`Linux version 4.4.35-1-pve (root@elsa) (gcc version 4.9.2 (Debian 4.9.2-10) ) #1 SMP Fri Dec 9 11:09:55 CET 2016`,
		},

		"Fedora 26": {
			`Linux version 4.12.13-300.fc26.x86_64 (mockbuild@bkernel01.phx2.fedoraproject.org) (gcc version 7.1.1 20170622 (Red Hat 7.1.1-3) (GCC) ) #1 SMP Thu Sep 14 16:00:38 UTC 2017`,
		},

		"Fedora 25": {
			`Linux version 4.11.12-200.fc25.x86_64 (mockbuild@bkernel01.phx2.fedoraproject.org) (gcc version 6.3.1 20161221 (Red Hat 6.3.1-1) (GCC) ) #1 SMP Fri Jul 21 16:41:43 UTC 2017`,
		},

		"CentOS": {
			`Linux version 3.10.0-327.22.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu Jun 23 17:05:11 UTC 2016`,
			`Linux version 3.10.0-327.18.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu May 12 11:03:55 UTC 2016`,
			`Linux version 3.10.0-327.28.3.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Thu Aug 18 19:05:49 UTC 2016`,
			`Linux version 3.10.0-327.36.3.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.5 20150623 (Red Hat 4.8.5-4) (GCC) ) #1 SMP Mon Oct 24 16:09:20 UTC 2016`,
			`Linux version 3.10.0-514.10.2.el7.x86_64 (builder@kbuilder.dev.centos.org) (gcc version 4.8.5 20150623 (Red Hat 4.8.5-11) (GCC) ) #1 SMP Fri Mar 3 00:04:05 UTC 2017`,
		},

		"Arch": {
			`Linux version 4.9.43-1-ARCH (builduser@leming) (gcc version 7.1.1 20170630 (GCC) ) #1 SMP Fri Aug 18 01:10:29 UTC 2017`,
		},

		// Docker for macOS
		"Moby": {
			`Linux version 4.9.41-moby (root@11fbdc1f630f) (gcc version 6.2.1 20160822 (Alpine 6.2.1) ) #1 SMP Wed Sep 6 00:05:16 UTC 2017`,
		},

		"Amazon": {
			`Linux version 4.9.38-16.35.amzn1.x86_64 (mockbuild@gobi-build-60006) (gcc version 4.8.3 20140911 (Red Hat 4.8.3-9) (GCC) ) #1 SMP Sat Aug 5 01:39:35 UTC 2017`,
		},

		"Microsoft": {
			`Linux version 4.4.0-43-Microsoft (Microsoft@Microsoft.com) (gcc version 5.4.0 (GCC) ) #1-Microsoft Wed Dec 31 14:42:53 PST 2014`,
		},

		"unknown": {
			``,
			`lalala`,
		},
	} {
		for _, v := range procVersion {
			actual := getLinuxDistribution(v)
			assert.Equal(t, expected, actual)
		}
	}
}
