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
	"net/url"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
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
	testVMAuth        = "http://victoriametrics_pmm:vm-password@pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
	testVMAuthNoCreds = "http://pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
	// A password lost from the secret, or a typo: half a credential.
	testVMAuthUsernameOnly = "http://victoriametrics_pmm@pmm-ha-vmauth.pmm.svc.cluster.local:8427/"
	testVMAuthWrite        = "http://pmm-ha-vmauth.pmm.svc.cluster.local:8427/api/v1/write"
	testInjectedURL        = "https://collector.example.com/api/v1/write"
	// A comma-separated list of remote-write URLs, the shape vmagent takes, where every element
	// carries its own credentials.
	testInjectedURLList = "https://collector:secret@collector.example.com/api/v1/write," +
		"https://mirror:other-secret@mirror.example.com/api/v1/write"
	testInjectedURLListRedacted = "https://<redacted>@collector.example.com/api/v1/write," +
		"https://<redacted>@mirror.example.com/api/v1/write"
)

func testLogger() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return logrus.NewEntry(l)
}

// clearVMAgentEnv removes every VMAGENT_* variable, and the PMM_* variables buildVMAgentProcess
// reads, from the test process for the duration of the test, so that a tuned developer
// environment cannot leak into the expected output.
func clearVMAgentEnv(t *testing.T) {
	t.Helper()
	unset := func(key string) {
		value, ok := os.LookupEnv(key)
		if !ok {
			return
		}
		// Setenv registers the restore; Unsetenv clears the variable for the test.
		t.Setenv(key, value)
		require.NoError(t, os.Unsetenv(key))
	}
	for _, env := range os.Environ() {
		if key, _, ok := strings.Cut(env, "="); ok && strings.HasPrefix(key, "VMAGENT_") {
			unset(key)
		}
	}
	unset(maxScrapeSizeEnv)
	unset("PMM_INTERFACE_TO_BIND")
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
	require.True(t, ok, "missing env %s in %v", key, env)
	assert.Equal(t, want, got, "env %s", key)
}

func assertNoEnv(t *testing.T, env []string, key string) {
	t.Helper()
	got, ok := envValue(env, key)
	assert.False(t, ok, "unexpected env %s='%s'", key, got)
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
	clearVMAgentEnv(t)

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

func TestVMAgentConfigDebugLog(t *testing.T) {
	clearVMAgentEnv(t)
	logger, hook := test.NewNullLogger()
	logger.SetLevel(logrus.DebugLevel)
	l := logrus.NewEntry(logger)

	t.Run("names the credential source, never the credential", func(t *testing.T) {
		hook.Reset()
		vmAgentConfig(l, "", newVMParams(t, testVMAuth), vmAgentDeployment{haEnabled: true, isServerAgent: true})
		require.Len(t, hook.Entries, 1)
		entry := hook.LastEntry()
		assert.Equal(t, credentialVMURL, entry.Data["credential_source"])
		assert.Equal(t, testVMAuthWrite, entry.Data["remote_write_url"])
		line, err := entry.String()
		require.NoError(t, err)
		assert.NotContains(t, line, "vm-password")
	})

	t.Run("an injected URL is logged without its userinfo", func(t *testing.T) {
		hook.Reset()
		t.Setenv(envRemoteWriteURL, "https://collector:secret@collector.example.com/api/v1/write")
		vmAgentConfig(l, "", newVMParams(t, models.VMBaseURL), vmAgentDeployment{})
		entry := hook.LastEntry()
		assert.Equal(t, credentialNone, entry.Data["credential_source"])
		assert.Equal(t, "https://<redacted>@collector.example.com/api/v1/write", entry.Data["remote_write_url"])
		line, err := entry.String()
		require.NoError(t, err)
		assert.NotContains(t, line, "secret")
	})

	t.Run("every element of an injected URL list is logged without its userinfo", func(t *testing.T) {
		hook.Reset()
		t.Setenv(envRemoteWriteURL, testInjectedURLList)
		vmAgentConfig(l, "", newVMParams(t, models.VMBaseURL), vmAgentDeployment{})
		entry := hook.LastEntry()
		assert.Equal(t, testInjectedURLListRedacted, entry.Data["remote_write_url"])
		line, err := entry.String()
		require.NoError(t, err)
		assert.NotContains(t, line, "secret")
		assert.NotContains(t, line, "other-secret")
	})

	t.Run("injected credentials are reported as the source", func(t *testing.T) {
		hook.Reset()
		t.Setenv(envRemoteWriteUsername, "injected-user")
		vmAgentConfig(l, "", newVMParams(t, models.VMBaseURL), vmAgentDeployment{})
		assert.Equal(t, credentialInjected, hook.LastEntry().Data["credential_source"])
	})
}

func TestBuildVMAgentProcess(t *testing.T) {
	clearVMAgentEnv(t)

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

	t.Run("half an injected pair is sent alone, not completed with PMM's other half", func(t *testing.T) {
		t.Setenv(envRemoteWriteUsername, "injected-user")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, "injected-user", "")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-pass")
	})

	t.Run("a bearer token or OAuth2 client withholds the default pair", func(t *testing.T) {
		// vmagent refuses to start with basic auth and either of these set.
		for _, name := range []string{"VMAGENT_remoteWrite_bearerToken", "VMAGENT_remoteWrite_oauth2_clientID"} {
			t.Run(name, func(t *testing.T) {
				t.Setenv(name, "value")
				actual := buildVMAgentProcess(testLogger(), "", pair)
				assertEnv(t, actual.Env, name, "value")
				assertCredentials(t, actual.Env, "", "")
				assertNotAnywhere(t, actual.Env, actual.Args, "default-pass")
			})
		}
	})

	t.Run("custom headers and a client certificate compose with the default pair", func(t *testing.T) {
		// A tenant header on top of the VM credential is a normal shape; the pair must stay.
		t.Setenv("VMAGENT_remoteWrite_headers", "X-Scope-OrgID:1")
		t.Setenv("VMAGENT_remoteWrite_tlsCertFile", "/run/secrets/client.crt")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, "VMAGENT_remoteWrite_headers", "X-Scope-OrgID:1")
		assertCredentials(t, actual.Env, "default-user", "default-pass")
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

	t.Run("injected URL with only a username withholds the default password", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		t.Setenv(envRemoteWriteUsername, "collector-user")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, "collector-user", "")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-pass")
	})

	t.Run("injected URL with only a password withholds the default username", func(t *testing.T) {
		t.Setenv(envRemoteWriteURL, testInjectedURL)
		t.Setenv(envRemoteWritePassword, "collector-pass")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, "", "collector-pass")
		assertNotAnywhere(t, actual.Env, actual.Args, "default-user")
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
		// The documented opt-in for redirecting writes to another PMM Server while keeping
		// per-client PMM credentials: injected values are rendered on the client like PMM's own.
		t.Setenv(envRemoteWriteURL, "https://pmm.example.com/victoriametrics/api/v1/write")
		t.Setenv(envRemoteWriteUsername, serverUsernameTmpl)
		t.Setenv(envRemoteWritePassword, serverPasswordTmpl)
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertCredentials(t, actual.Env, serverUsernameTmpl, serverPasswordTmpl)
	})

	t.Run("empty injected variables are ignored", func(t *testing.T) {
		// vmagent exits on an empty URL and treats an empty credential as no authentication, so an
		// empty variable is not forwarded and does not displace PMM's default; ParseEnvVars warns.
		t.Setenv(envRemoteWriteURL, "")
		t.Setenv(envRemoteWriteUsername, "")
		t.Setenv(envRemoteWritePassword, "")
		t.Setenv("VMAGENT_loggerLevel", "")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "default-user", "default-pass")
		assertEnv(t, actual.Env, "VMAGENT_loggerLevel", "INFO")
		assert.Len(t, actual.Env, 7)
	})

	t.Run("a pair without credentials emits none", func(t *testing.T) {
		actual := buildVMAgentProcess(testLogger(), "", remoteWrite{url: pair.url, source: credentialNone})
		assertEnv(t, actual.Env, envRemoteWriteURL, pair.url)
		assertCredentials(t, actual.Env, "", "")
		assert.Len(t, actual.Env, 5)
	})

	t.Run("unrelated VMAGENT_ variables pass through, other prefixes and cases do not", func(t *testing.T) {
		// Kubernetes service links inject upper-case VMAGENT_* names into HA pods (the VictoriaMetrics
		// operator names its Service vmagent-<name>); they are inert for vmagent and must neither be
		// dropped nor confused with routing variables. The prefix match is case-sensitive, like vmagent's.
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		t.Setenv("VMAGENT_PMM_HA_VMAGENT_PORT_8429_TCP", "tcp://10.96.0.1:8429")
		t.Setenv("VM_retentionPeriod", "30d")
		t.Setenv("vmagent_loggerLevel", "DEBUG")
		actual := buildVMAgentProcess(testLogger(), "", pair)
		assertEnv(t, actual.Env, "VMAGENT_PMM_HA_VMAGENT_SERVICE_HOST", "10.96.0.1")
		assertEnv(t, actual.Env, "VMAGENT_PMM_HA_VMAGENT_PORT_8429_TCP", "tcp://10.96.0.1:8429")
		assertNoEnv(t, actual.Env, "VM_retentionPeriod")
		assertNoEnv(t, actual.Env, "vmagent_loggerLevel")
		assertEnv(t, actual.Env, "VMAGENT_loggerLevel", "INFO")
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
			name: "password only",
			url:  "http://:vmpass@victoriametrics:8428",
			want: remoteWrite{url: testExternalVMWrite, password: "vmpass", source: credentialVMURL},
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
			u, err := url.Parse(tc.url)
			require.NoError(t, err)
			assert.Equal(t, tc.want, vmRemoteWrite(u))
		})
	}
}

func TestSplitUserinfo(t *testing.T) {
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
		{name: "password only", url: "http://:vmpass@victoriametrics:8428", expectedURL: testExternalVM, expectedPassword: "vmpass"},
		{name: "special characters", url: "http://us%40er:p%40ss%3A1@victoriametrics:8428", expectedURL: testExternalVM, expectedUsername: "us@er", expectedPassword: "p@ss:1"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := url.Parse(tc.url)
			require.NoError(t, err)
			gotURL, gotUsername, gotPassword := splitUserinfo(u)
			assert.Equal(t, tc.expectedURL, gotURL.String())
			assert.Equal(t, tc.expectedUsername, gotUsername)
			assert.Equal(t, tc.expectedPassword, gotPassword)
			// The input is left intact: callers hold the shared params URL.
			assert.Equal(t, tc.url, u.String())
		})
	}
}
