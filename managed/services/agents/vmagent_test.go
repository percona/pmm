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

package agents

import (
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	inventoryv1 "github.com/percona/pmm/api/inventory/v1"
	"github.com/percona/pmm/managed/models"
)

// Fixture URLs shared by the vmagent tests. Credentials in them are fixtures, not secrets.
const (
	testExternalVM      = "http://victoriametrics:8428"
	testExternalVMWrite = "http://victoriametrics:8428/api/v1/write"
	testExternalVMAuth  = "http://vmuser:vmpass@victoriametrics:8428"
	// The shape the pmm-ha chart composes PMM_VM_URL in: vmauth credentials in the userinfo.
	testVMAuth      = "http://victoriametrics_pmm:vm-password@pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
	testVMAuthWrite = "http://pmm-ha-vmauth.pmm.svc.cluster.local:8427/api/v1/write"
	testInjectedURL = "https://collector.example.com/api/v1/write"
)

func testLogger() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return logrus.NewEntry(l)
}

func newVMParams(t *testing.T, vmURL string) *models.VictoriaMetricsParams {
	t.Helper()
	params, err := models.NewVictoriaMetricsParams(models.BasePrometheusConfigPath, vmURL)
	require.NoError(t, err)
	return params
}

// envValue returns the value of key in a KEY=value slice.
func envValue(env []string, key string) (string, bool) {
	for _, e := range env {
		if k, v, ok := strings.Cut(e, "="); ok && k == key {
			return v, true
		}
	}
	return "", false
}

func assertEnv(t *testing.T, env []string, key, want string) {
	t.Helper()
	got, ok := envValue(env, key)
	require.True(t, ok, "missing env %q in %v", key, env)
	assert.Equal(t, want, got, "env %q", key)
}

func assertNoEnv(t *testing.T, env []string, key string) {
	t.Helper()
	got, ok := envValue(env, key)
	assert.False(t, ok, "unexpected env %s=%q", key, got)
}

// assertCredentials asserts the basic-auth pair emitted for the vmagent; an empty value asserts
// that the corresponding variable is absent.
func assertCredentials(t *testing.T, env []string, username, password string) {
	t.Helper()
	if username == "" {
		assertNoEnv(t, env, envRemoteWriteUsername)
	} else {
		assertEnv(t, env, envRemoteWriteUsername, username)
	}
	if password == "" {
		assertNoEnv(t, env, envRemoteWritePassword)
	} else {
		assertEnv(t, env, envRemoteWritePassword, password)
	}
}

// assertNoServerCredentialTemplates asserts that no env entry would render to the client's PMM
// Server credentials.
func assertNoServerCredentialTemplates(t *testing.T, env []string) {
	t.Helper()
	for _, e := range env {
		assert.NotContains(t, e, serverUsernameTmpl)
		assert.NotContains(t, e, serverPasswordTmpl)
	}
}

// assertNotAnywhere asserts that value appears in no env entry and no argument, whatever the key.
func assertNotAnywhere(t *testing.T, env, args []string, value string) {
	t.Helper()
	for _, e := range env {
		assert.NotContains(t, e, value)
	}
	for _, a := range args {
		assert.NotContains(t, a, value)
	}
}

func TestVMAgentConfigSelectsPath(t *testing.T) {
	// The dispatcher is the only place that looks at the deployment mode.
	t.Run("standalone ignores the server-agent flag", func(t *testing.T) {
		params := newVMParams(t, models.VMBaseURL)
		client := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{})
		server := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{isServerAgent: true})
		assert.Equal(t, client.Env, server.Env)
		assertEnv(t, client.Env, envRemoteWriteURL, serverProxyWriteURL)
	})

	t.Run("HA distinguishes clients from the server agent", func(t *testing.T) {
		params := newVMParams(t, testVMAuth)
		client := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{haEnabled: true})
		server := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{haEnabled: true, isServerAgent: true})
		assertEnv(t, client.Env, envRemoteWriteURL, serverProxyWriteURL)
		assertEnv(t, server.Env, envRemoteWriteURL, testVMAuthWrite)
	})

	t.Run("the same off-box VM URL routes clients differently in standalone and HA", func(t *testing.T) {
		// Standalone: an off-box VM is an external VM the operator made reachable, write directly.
		// HA: the off-box VM is vmauth, reachable only in-cluster, so clients write via the server.
		params := newVMParams(t, testVMAuth)
		standalone := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{})
		ha := vmAgentConfig(testLogger(), "", params, vmAgentDeployment{haEnabled: true})
		assertEnv(t, standalone.Env, envRemoteWriteURL, testVMAuthWrite)
		assertEnv(t, ha.Env, envRemoteWriteURL, serverProxyWriteURL)
	})
}

func TestBuildVMAgentProcess(t *testing.T) {
	// A neutral pair: these tests are about layering the operator's environment on top of whatever
	// pair a path picked, not about any particular path.
	pair := remoteWrite{
		url:      "http://vm.example:8428/api/v1/write",
		username: "default-user",
		password: "default-pass",
		source:   credentialVMURL,
	}

	t.Run("canonical args, credentials never on the command line", func(t *testing.T) {
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assert.Equal(t, []string{
			"-envflag.enable=true",
			"-envflag.prefix=VMAGENT_",
			"-httpListenAddr=127.0.0.1:{{.listen_port}}",
			"-promscrape.config={{.TextFiles.vmagentscrapecfg}}",
			"-remoteWrite.tmpDataPath={{.tmp_dir}}/vmagent-temp-dir",
		}, actual.Args)
		assertNotAnywhere(t, nil, actual.Args, "basicAuth")
		assertNotAnywhere(t, nil, actual.Args, "default-pass")
	})

	t.Run("bind interface follows PMM_INTERFACE_TO_BIND", func(t *testing.T) {
		t.Setenv("PMM_INTERFACE_TO_BIND", "0.0.0.0")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assert.Contains(t, actual.Args, "-httpListenAddr=0.0.0.0:{{.listen_port}}")
	})

	t.Run("defaults", func(t *testing.T) {
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assert.Equal(t, inventoryv1.AgentType_AGENT_TYPE_VM_AGENT, actual.Type)
		assert.Equal(t, "{{", actual.TemplateLeftDelim)
		assert.Equal(t, "}}", actual.TemplateRightDelim)
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "default-user", "default-pass")
		assertEnv(t, actual.Env, "VMAGENT_remoteWrite_tlsInsecureSkipVerify", "{{.server_insecure}}")
		assertEnv(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize", maxScrapeSizeDefault)
		assertEnv(t, actual.Env, "VMAGENT_remoteWrite_maxDiskUsagePerURL", "1073741824")
		assertEnv(t, actual.Env, "VMAGENT_loggerLevel", "INFO")
		assert.Len(t, actual.Env, 7)
	})

	t.Run("PMM_PROMSCRAPE_MAX_SCRAPE_SIZE sets the scrape size default", func(t *testing.T) {
		t.Setenv(maxScrapeSizeEnv, "16MiB")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize", "16MiB")
	})

	t.Run("injected tuning variables win over defaults", func(t *testing.T) {
		t.Setenv(maxScrapeSizeEnv, "16MiB")
		t.Setenv("VMAGENT_promscrape_maxScrapeSize", "32MiB")
		t.Setenv("VMAGENT_loggerLevel", "DEBUG")
		t.Setenv("VMAGENT_remoteWrite_maxDiskUsagePerURL", "52428800")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, "VMAGENT_promscrape_maxScrapeSize", "32MiB")
		assertEnv(t, actual.Env, "VMAGENT_loggerLevel", "DEBUG")
		assertEnv(t, actual.Env, "VMAGENT_remoteWrite_maxDiskUsagePerURL", "52428800")
		assert.Len(t, actual.Env, 7)
	})

	t.Run("injected credentials win over the default pair", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "injected-user")
		t.Setenv(envRemoteWritePassword, "injected-pass")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "injected-user", "injected-pass")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-pass")
	})

	t.Run("a partially injected credential keeps the other default", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "injected-user")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, "injected-user", "default-pass")
	})

	t.Run("injected URL withholds the default credential", func(t *testing.T) {
		// PMM's default credential belongs to PMM's default endpoint; an operator-chosen endpoint
		// gets nothing PMM derived.
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, actual.Env, "", "")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-user")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-pass")
	})

	t.Run("injected URL and credentials travel together", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		t.Setenv(envRemoteWriteUsername, "injected-user")
		t.Setenv(envRemoteWritePassword, "injected-pass")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, envRemoteWriteURL, testInjectedURL)
		assertCredentials(t, actual.Env, "injected-user", "injected-pass")
	})

	t.Run("credential templates injected alongside a URL pass through verbatim", func(t *testing.T) {
		// The documented opt-in for redirecting writes while keeping per-client PMM credentials:
		// injected values are rendered on the client like PMM's own defaults.
		t.Setenv(envRemoteWriteURL, "https://pmm.example.com/victoriametrics/api/v1/write")
		t.Setenv(envRemoteWriteUsername, serverUsernameTmpl)
		t.Setenv(envRemoteWritePassword, serverPasswordTmpl)
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, serverUsernameTmpl, serverPasswordTmpl)
	})

	t.Run("a pair without credentials emits none", func(t *testing.T) {
		actual := buildVMAgentProcess(testLogger(), "", remoteWrite{url: pair.url, source: credentialNone})
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "", "")
		assert.Len(t, actual.Env, 5)
	})

	t.Run("unrelated VMAGENT_ variables pass through, other prefixes do not", func(t *testing.T) {
		// Kubernetes service links inject upper-case VMAGENT_* names into HA pods; they are inert
		// for vmagent and must neither be dropped nor confused with routing variables.
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_PORT_8429_TCP", "tcp://10.96.0.1:8429")
		t.Setenv("VM_retentionPeriod", "30d")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, "VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		assertEnv(t, actual.Env, "VMAGENT_PMM_HA_VMAGENT_PORT_8429_TCP", "tcp://10.96.0.1:8429")
		assertNoEnv(t, actual.Env, "VM_retentionPeriod")
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "default-user", "default-pass")
	})

	t.Run("env is sorted and deterministic", func(t *testing.T) {
		t.Setenv("VMAGENT_loggerLevel", "WARN")
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		first := buildVMAgentProcess(testLogger(), "", pair)
		second := buildVMAgentProcess(testLogger(), "", pair)
		assert.True(t, sort.StringsAreSorted(first.Env), "%v", first.Env)
		assert.Equal(t, first.Env, second.Env)
	})

	t.Run("scrape config is passed as a text file", func(t *testing.T) {
		actual := buildVMAgentProcess(testLogger(), "scrape_configs: []", pair)
		assert.Equal(t, map[string]string{"vmagentscrapecfg": "scrape_configs: []"}, actual.TextFiles)
	})
}

func TestVMRemoteWrite(t *testing.T) {
	testCases := []struct {
		name string
		url  string
		want remoteWrite
	}{
		{
			name: "no credentials, trailing slash",
			url:  "http://victoriametrics:8428/",
			want: remoteWrite{url: testExternalVMWrite, source: credentialNone},
		},
		{
			name: "no credentials, no trailing slash",
			url:  testExternalVM,
			want: remoteWrite{url: testExternalVMWrite, source: credentialNone},
		},
		{
			name: "path prefix is kept",
			url:  "https://vm.example.com/victoria/",
			want: remoteWrite{url: "https://vm.example.com/victoria/api/v1/write", source: credentialNone},
		},
		{
			name: "credentials move from the URL into the pair",
			url:  testExternalVMAuth,
			want: remoteWrite{url: testExternalVMWrite, username: "vmuser", password: "vmpass", source: credentialVMURL},
		},
		{
			name: "username only",
			url:  "http://vmuser@victoriametrics:8428",
			want: remoteWrite{url: testExternalVMWrite, username: "vmuser", source: credentialVMURL},
		},
		{
			name: "percent-encoded credentials are decoded",
			url:  "http://us%40er:p%40ss%3A1@victoriametrics:8428",
			want: remoteWrite{url: testExternalVMWrite, username: "us@er", password: "p@ss:1", source: credentialVMURL},
		},
		{
			name: "the chart's vmauth URL",
			url:  testVMAuth,
			want: remoteWrite{url: testVMAuthWrite, username: "victoriametrics_pmm", password: "vm-password", source: credentialVMURL},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, vmRemoteWrite(tc.url))
		})
	}
}

func TestSplitURLCredentials(t *testing.T) {
	testCases := []struct {
		name             string
		url              string
		expectedURL      string
		expectedUsername string
		expectedPassword string
	}{
		{name: "empty", url: "", expectedURL: ""},
		{name: "no credentials", url: "http://victoriametrics:8428/", expectedURL: "http://victoriametrics:8428/"},
		{name: "username and password", url: testExternalVMAuth, expectedURL: testExternalVM, expectedUsername: "vmuser", expectedPassword: "vmpass"},
		{name: "username only", url: "http://vmuser@victoriametrics:8428", expectedURL: testExternalVM, expectedUsername: "vmuser"},
		{name: "empty password", url: "http://vmuser:@victoriametrics:8428", expectedURL: testExternalVM, expectedUsername: "vmuser"},
		{name: "special characters", url: "http://us%40er:p%40ss%3A1@victoriametrics:8428", expectedURL: testExternalVM, expectedUsername: "us@er", expectedPassword: "p@ss:1"},
		{name: "unparsable URL is returned unchanged", url: "://not-a-url", expectedURL: "://not-a-url"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotURL, gotUsername, gotPassword := splitURLCredentials(tc.url)
			assert.Equal(t, tc.expectedURL, gotURL)
			assert.Equal(t, tc.expectedUsername, gotUsername)
			assert.Equal(t, tc.expectedPassword, gotPassword)
		})
	}
}
