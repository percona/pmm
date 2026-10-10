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

package management

import (
	"encoding/json"
	"fmt"
	"maps"
	"net"
	"slices"
	"strconv"
	"strings"

	"github.com/percona/pmm/admin/agentlocal"
	"github.com/percona/pmm/admin/commands"
	"github.com/percona/pmm/admin/pkg/flags"
	inventoryClient "github.com/percona/pmm/api/inventory/v1/json/client"
	services "github.com/percona/pmm/api/inventory/v1/json/client/services_service"
	"github.com/percona/pmm/api/management/v1/json/client"
	mservice "github.com/percona/pmm/api/management/v1/json/client/management_service"
)

const updateHelp = `Changes the Service in place: the Service and its Agents keep their IDs, metrics and query
analytics history, and backups.

Only the flags you pass are changed; everything else stays as it is. The defaults shown below
apply to "pmm-admin add" only. Use --no-<flag> or --<flag>=false to turn an option off, and an
empty value, e.g. --disable-collectors= or --custom-labels=, to clear a list.`

// UpdateCommand is used by Kong for CLI flags and commands.
type UpdateCommand struct {
	External           UpdateExternalCommand           `cmd:"" help:"Update External source of data in place (only passed flags are changed)"`
	ExternalServerless UpdateExternalServerlessCommand `cmd:"" help:"Update External Service on Remote node in place (only passed flags are changed)"`
	HAProxy            UpdateHAProxyCommand            `cmd:"" name:"haproxy" help:"Update HAProxy in place (only passed flags are changed)"`
	MongoDB            UpdateMongoDBCommand            `cmd:"" name:"mongodb" help:"Update MongoDB in place (only passed flags are changed)"`
	MySQL              UpdateMySQLCommand              `cmd:"" name:"mysql" help:"Update MySQL in place (only passed flags are changed)"`
	PostgreSQL         UpdatePostgreSQLCommand         `cmd:"" name:"postgresql" help:"Update PostgreSQL in place (only passed flags are changed)"`
	Valkey             UpdateValkeyCommand             `cmd:"" name:"valkey" help:"Update Valkey in place (only passed flags are changed)"`
	ProxySQL           UpdateProxySQLCommand           `cmd:"" name:"proxysql" help:"Update ProxySQL in place (only passed flags are changed)"`
}

// UpdateCommonFlags is used by Kong for the flags shared by all update commands.
type UpdateCommonFlags struct {
	ServiceID string `help:"ID of the Service to update (default: the Service with the given name)"`
	DryRun    bool   `help:"Show what would change, without changing anything"`

	// typed holds the names of the flags and arguments given on the command line.
	typed map[string]bool
}

// SetTypedFlags records the flags and arguments given on the command line; update changes only those.
func (f *UpdateCommonFlags) SetTypedFlags(names map[string]bool) {
	f.typed = names
}

// rejectTyped fails if any of the given flags is passed, as update cannot change them.
func (f *UpdateCommonFlags) rejectTyped(names ...string) error {
	for _, name := range names {
		if f.typed[name] {
			return fmt.Errorf("--%s cannot be changed by update; remove the Service and add it again", name)
		}
	}

	return nil
}

func (f *UpdateCommonFlags) str(name, value string) *string {
	if !f.typed[name] {
		return nil
	}

	return &value
}

func (f *UpdateCommonFlags) boolean(name string, value bool) *bool {
	if !f.typed[name] {
		return nil
	}

	return &value
}

func (f *UpdateCommonFlags) num32(name string, value int32) *int32 {
	if !f.typed[name] {
		return nil
	}

	return &value
}

// file returns the content of the file given by the flag; an empty path clears the stored content.
func (f *UpdateCommonFlags) file(name, path string) (*string, error) {
	if !f.typed[name] {
		return nil, nil //nolint:nilnil
	}

	content, err := commands.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return &content, nil
}

// labels returns the parsed labels given by the flag; an empty value clears them.
func (f *UpdateCommonFlags) labels(name string, value map[string]string) map[string]string {
	if !f.typed[name] {
		return nil
	}

	return *commands.ParseKeyValuePair(&value)
}

// list returns the trimmed list given by the flag; an empty value clears it.
func (f *UpdateCommonFlags) list(name string, value []string) []string {
	if !f.typed[name] {
		return nil
	}

	return append([]string{}, commands.ParseDisableCollectors(value)...)
}

func (f *UpdateCommonFlags) metricsMode(value flags.MetricsMode) *string {
	if !f.typed["metrics-mode"] {
		return nil
	}

	// "auto" is the unspecified mode in the API; its own enum name would be dropped as unknown.
	if value == "auto" {
		return new("METRICS_MODE_UNSPECIFIED")
	}

	return value.EnumValue()
}

func (f *UpdateCommonFlags) logLevel(value flags.LogLevel) *string {
	if !f.typed["log-level"] {
		return nil
	}

	return value.EnumValue()
}

func (f *UpdateCommonFlags) connectionTimeout(value string) string {
	if !f.typed["connection-timeout"] {
		return ""
	}

	return value
}

// credentials returns the credentials given by the flags or read from --credentials-source,
// where only the values present in the file are changed.
func (f *UpdateCommonFlags) credentials(username, password, agentPassword, source string) (*string, *string, *string, error) {
	u, p, a := f.str("username", username), f.str("password", password), f.str("agent-password", agentPassword)
	if !f.typed["credentials-source"] {
		return u, p, a, nil
	}

	creds, err := commands.ReadFromSource(source)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to retrieve credentials from %s: %w", source, err)
	}

	if creds.Username != "" {
		u = &creds.Username
	}
	if creds.Password != "" {
		p = &creds.Password
	}
	if creds.AgentPassword != "" {
		a = &creds.AgentPassword
	}

	return u, p, a, nil
}

// endpoint returns the address, port and socket given by the address argument and the --host, --port and
// --socket flags.
func (f *UpdateCommonFlags) endpoint(address string, common AddCommonFlags, socket string) (*string, *int64, *string, error) {
	var host *string
	var port *int64
	if f.typed["address"] {
		h, p, err := net.SplitHostPort(address)
		if err != nil {
			return nil, nil, nil, err
		}

		portI, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return nil, nil, nil, err
		}

		host, port = &h, &portI
	}

	if f.typed["host"] {
		host = &common.AddHostFlag
	}
	if f.typed["port"] {
		port = new(int64(common.AddPortFlag))
	}

	return host, port, f.str("socket", socket), nil
}

// target returns the ID or name of the Service to update: --service-id, the name given on the command line,
// or else the only Service of the type on this Node.
func (f *UpdateCommonFlags) target(name string, serviceType string) (string, error) {
	if f.ServiceID != "" {
		return f.ServiceID, nil
	}

	if name != "" {
		return name, nil
	}

	status, err := agentlocal.GetStatus(agentlocal.DoNotRequestNetworkInfo)
	if err != nil {
		return "", err
	}

	res, err := inventoryClient.Default.ServicesService.ListServices(&services.ListServicesParams{
		NodeID:      &status.NodeID,
		ServiceType: &serviceType,
		Context:     commands.Ctx,
	})
	if err != nil {
		return "", err
	}

	p := res.Payload
	var ids []string
	for _, s := range p.Mysql {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.Mongodb {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.Postgresql {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.Proxysql {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.Haproxy {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.External {
		ids = append(ids, s.ServiceID)
	}
	for _, s := range p.Valkey {
		ids = append(ids, s.ServiceID)
	}

	if len(ids) != 1 {
		return "", fmt.Errorf("found %d Services of this type on this Node; pass the Service name or --service-id", len(ids))
	}

	return ids[0], nil
}

// typedValue returns the value of the flag if it is given on the command line, and an empty string otherwise.
func (f *UpdateCommonFlags) typedValue(name, value string) string {
	if !f.typed[name] {
		return ""
	}

	return value
}

// typedName returns the Service name given by the name argument or the --service-name flag, if any.
func (f *UpdateCommonFlags) typedName(name string, common AddCommonFlags) string {
	switch {
	case f.typed["service-name"]:
		return common.AddServiceNameFlag
	case f.typed["name"]:
		return name
	default:
		return ""
	}
}

// setting is a Service setting in the form of the "pmm-admin add" flag that sets it.
type setting struct {
	Flag  string `json:"flag"`
	Value string `json:"value"`
}

// settingChange is a setting changed by the update.
type settingChange struct {
	Flag   string `json:"flag"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// secretValue stands for a secret that is set.
const secretValue = "***"

var updateServiceResultT = commands.ParseTemplate(`
{{ .Title }}
Service ID  : {{ .ServiceID }}
Service name: {{ .ServiceName }}
{{- if .Warning }}

Warning: {{ .Warning }}
{{- end }}

{{ if .Changes }}{{ if .DryRun }}Settings the update would change:{{ else }}Changed settings:{{ end }}
{{- range .Changes }}
  {{ .Flag }}: {{ printf "%q" .Before }} -> {{ printf "%q" .After }}
{{- end }}
{{- else }}No settings changed.{{ end }}

Current settings:
{{- range .Settings }}
  {{ .Flag }}={{ .Value }}
{{- end }}
`)

type updateServiceResult struct {
	ServiceID   string          `json:"service_id"`
	ServiceName string          `json:"service_name"`
	DryRun      bool            `json:"dry_run"`
	Changed     bool            `json:"changed"`
	Changes     []settingChange `json:"changes"`
	Settings    []setting       `json:"settings"`
	Warning     string          `json:"warning,omitempty"`

	serviceTypeName string
}

func (res *updateServiceResult) Result() {}

// Title returns the first line of the text output.
func (res *updateServiceResult) Title() string {
	if res.DryRun {
		return res.serviceTypeName + " Service not changed (dry run)."
	}

	return res.serviceTypeName + " Service updated."
}

func (res *updateServiceResult) String() string {
	return commands.RenderTemplate(updateServiceResultT, res)
}

// serviceSettings returns the settings of the Service as flags.
type serviceSettings func(svc *mservice.UpdateServiceOKBodyAfter) []setting

// runUpdate sends the update and describes its result. The secretFlags map lists for each setting showing
// a secret the flags that change it, since the old and new secret look the same.
func (f *UpdateCommonFlags) runUpdate(
	target string, body mservice.UpdateServiceBody, serviceTypeName string, settings serviceSettings, secretFlags map[string][]string,
) (commands.Result, error) {
	body.DryRun = f.DryRun
	resp, err := client.Default.ManagementService.UpdateService(&mservice.UpdateServiceParams{
		ServiceID: target,
		Body:      body,
		Context:   commands.Ctx,
	})
	if err != nil {
		return nil, err
	}

	// Before and after share their schema, so before is read into the type of after to describe both alike.
	b, err := json.Marshal(resp.Payload.Before)
	if err != nil {
		return nil, err
	}
	var before mservice.UpdateServiceOKBodyAfter
	err = json.Unmarshal(b, &before)
	if err != nil {
		return nil, err
	}
	after := resp.Payload.After

	res := &updateServiceResult{
		ServiceID:       after.ServiceID,
		ServiceName:     after.ServiceName,
		DryRun:          f.DryRun,
		Settings:        settings(after),
		Warning:         resp.Payload.Warning,
		serviceTypeName: serviceTypeName,
	}

	beforeSettings := settings(&before)
	for _, s := range res.Settings {
		var old string
		for _, bs := range beforeSettings {
			if bs.Flag == s.Flag {
				old = bs.Value
			}
		}

		secretChanged := slices.ContainsFunc(secretFlags[s.Flag], func(name string) bool { return f.typed[name] })
		if old != s.Value || secretChanged {
			res.Changes = append(res.Changes, settingChange{Flag: s.Flag, Before: old, After: s.Value})
		}
	}
	res.Changed = len(res.Changes) != 0

	return res, nil
}

// agentOfType returns the first Agent of the given type, or an empty one if there is none.
func agentOfType(svc *mservice.UpdateServiceOKBodyAfter, agentTypes ...string) *mservice.UpdateServiceOKBodyAfterAgentsItems0 {
	for _, agent := range svc.Agents {
		if slices.Contains(agentTypes, agent.AgentType) {
			return agent
		}
	}

	return &mservice.UpdateServiceOKBodyAfterAgentsItems0{}
}

// querySource returns the query source of the Service from its QAN Agent types, keyed by query source names.
func querySource(svc *mservice.UpdateServiceOKBodyAfter, sources map[string]string) string {
	for _, agent := range svc.Agents {
		for source, agentType := range sources {
			if agent.AgentType == agentType {
				return source
			}
		}
	}

	return "none"
}

// labelsSettingValue formats labels the way the --custom-labels flag takes them.
func labelsSettingValue(labels map[string]string) string {
	pairs := make([]string, 0, len(labels))
	for _, k := range slices.Sorted(maps.Keys(labels)) {
		pairs = append(pairs, k+"="+labels[k])
	}

	return strings.Join(pairs, ",")
}

func secretSettingValue(isSet bool) string {
	if isSet {
		return secretValue
	}

	return ""
}

func boolSettingValue(value bool) string {
	return strconv.FormatBool(value)
}

func logLevelSettingValue(value *string) string {
	return strings.ToLower(strings.TrimPrefix(pointerValue(value), "LOG_LEVEL_"))
}

func metricsModeSettingValue(push bool) string {
	if push {
		return "push"
	}

	return "pull"
}

func pointerValue(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

// commonServiceSettings returns the labels of the Service and the settings shared by all exporters.
func commonServiceSettings(svc *mservice.UpdateServiceOKBodyAfter, exporter *mservice.UpdateServiceOKBodyAfterAgentsItems0) []setting {
	return []setting{
		{"--environment", svc.Environment},
		{"--cluster", svc.Cluster},
		{"--replication-set", svc.ReplicationSet},
		{"--custom-labels", labelsSettingValue(svc.CustomLabels)},
		{"--username", exporter.Username},
		{"--password", secretSettingValue(exporter.IsPasswordSet)},
		{"--metrics-mode", metricsModeSettingValue(exporter.PushMetrics)},
	}
}

// databaseServiceSettings returns the settings shared by the exporters of database Services.
func databaseServiceSettings(svc *mservice.UpdateServiceOKBodyAfter, exporter *mservice.UpdateServiceOKBodyAfterAgentsItems0) []setting {
	return append(
		commonServiceSettings(svc, exporter),
		setting{"--host", svc.Address},
		setting{"--port", portSettingValue(svc.Port)},
		setting{"--socket", svc.Socket},
		setting{"--agent-password", secretSettingValue(exporter.IsAgentPasswordSet)},
		setting{"--tls", boolSettingValue(exporter.TLS)},
		setting{"--tls-skip-verify", boolSettingValue(exporter.TLSSkipVerify)},
		setting{"--log-level", logLevelSettingValue(exporter.LogLevel)},
		setting{"--expose-exporter", boolSettingValue(exporter.ExposeExporter)},
		setting{"--connection-timeout", exporter.ConnectionTimeout},
	)
}

func portSettingValue(port int64) string {
	if port == 0 {
		return ""
	}

	return strconv.FormatInt(port, 10)
}

// credentialSecretFlags maps the secret settings of all Services to the flags that change them.
var credentialSecretFlags = map[string][]string{
	"--password":       {"password", "credentials-source"},
	"--agent-password": {"agent-password", "credentials-source"},
}
