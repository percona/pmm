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

package models

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	config "github.com/percona/promconfig"
	"gopkg.in/yaml.v3"
)

const (
	// BasePrometheusConfigPath - basic path with prometheus config,
	// that user can mount to container.
	BasePrometheusConfigPath = "/srv/prometheus/prometheus.base.yml"
	// VMBaseURL is the base URL for VictoriaMetrics.
	VMBaseURL = "http://127.0.0.1:9090/prometheus/"
)

// VictoriaMetricsParams - defines flags and settings for victoriametrics.
type VictoriaMetricsParams struct {
	// VMAlertFlags additional flags for VMAlert.
	VMAlertFlags []string
	// BaseConfigPath defines path for basic prometheus config.
	BaseConfigPath string
	// url defines url of Victoria Metrics
	url *url.URL
}

// ParseVictoriaMetricsURL parses and validates a VictoriaMetrics base URL (PMM_VM_URL): an http or
// https URL with a host. A trailing slash is appended when missing so that paths resolve under it.
// Error messages never echo credentials the URL may carry.
func ParseVictoriaMetricsURL(vmURL string) (*url.URL, error) {
	if !strings.HasSuffix(vmURL, "/") {
		vmURL += "/"
	}

	URL, err := url.Parse(vmURL)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("invalid VictoriaMetrics URL: %w", err)
	}
	if (URL.Scheme != "http" && URL.Scheme != "https") || URL.Host == "" || URL.Opaque != "" {
		// A scheme-less value is exactly the shape that lands here, and url.Parse leaves its
		// credentials in Opaque, where clearing URL.User would not reach them.
		return nil, fmt.Errorf("invalid VictoriaMetrics URL '%s': expected http(s)://host[:port][/path]", RedactURLCredentials(vmURL))
	}

	return URL, nil
}

// RedactURLCredentials replaces the userinfo of every URL in value with <redacted>, for URLs that
// reach a log line or an error message. The value may be a comma-separated list, the shape vmagent
// takes for a remote-write URL, and each element is redacted on its own: parsing the list as a
// single URL leaves everything after the first comma in the path, where the userinfo of the
// remaining elements survives untouched.
//
// A comma is also legal inside userinfo, so the split alone cannot tell a list of URLs from one
// URL whose password contains a comma. The split is therefore only trusted for a value that is a
// list of URLs throughout; anything else is redacted as the single URL it is, and is redacted
// whole when even that cannot be parsed or leaves an '@' behind.
func RedactURLCredentials(value string) string {
	elements := strings.Split(value, ",")
	if len(elements) > 1 && everyElementHasScheme(elements) {
		for i, element := range elements {
			redacted, ok := redactURLElementCredentials(element)
			if !ok && strings.Contains(element, "@") {
				redacted = "<redacted>"
			}
			elements[i] = redacted
		}

		return strings.Join(elements, ",")
	}

	// Not a list, so any comma belongs to this one URL and it is redacted as one. Splitting first
	// would hand the text before the comma to an element of its own, where the front of a password
	// no longer looks like a credential and would be printed verbatim.
	redacted, ok := redactURLElementCredentials(value)
	if !ok && strings.Contains(value, "@") {
		return "<redacted>"
	}
	// An '@' left after a comma means this was a list after all, malformed enough that parsing it
	// as a single URL left a later element's userinfo in the path untouched.
	comma := strings.Index(redacted, ",")
	if comma >= 0 && strings.Contains(redacted[comma:], "@") {
		return "<redacted>"
	}

	return redacted
}

// everyElementHasScheme reports whether each element carries a scheme, which is what vmagent
// requires of a remote-write URL and what makes the comma that separated them a list separator
// rather than part of a credential. An element that carries a scheme but does not parse still
// belongs to the list and is redacted on its own; an empty element carries nothing and does not
// disqualify the list.
func everyElementHasScheme(elements []string) bool {
	for _, element := range elements {
		if element != "" && !strings.Contains(element, "://") {
			return false
		}
	}

	return true
}

// redactURLElementCredentials redacts one URL and reports whether it could be parsed at all. A
// scheme-less user:pass@host parses as an opaque URL with no userinfo to drop, so it is parsed as
// an authority instead. A value that does not parse cannot be split, and the caller redacts it
// rather than guess what the unparsed text holds.
func redactURLElementCredentials(value string) (string, bool) {
	schemeless := !strings.Contains(value, "://")
	raw := value
	if schemeless {
		raw = "//" + value
	}
	u, err := url.Parse(raw)
	if err != nil {
		return value, false
	}
	if u.User == nil {
		return value, true
	}
	u.User = nil
	stripped := u.String()
	if schemeless {
		return "<redacted>@" + strings.TrimPrefix(stripped, "//"), true
	}
	scheme := strings.Index(stripped, "://")
	if scheme >= 0 {
		return stripped[:scheme+3] + "<redacted>@" + stripped[scheme+3:], true
	}

	return stripped, true
}

// NewVictoriaMetricsParams - returns configuration params for VictoriaMetrics.
func NewVictoriaMetricsParams(basePath, vmURL string) (*VictoriaMetricsParams, error) {
	URL, err := ParseVictoriaMetricsURL(vmURL)
	if err != nil {
		return nil, err
	}
	vmp := &VictoriaMetricsParams{
		BaseConfigPath: basePath,
		url:            URL,
	}
	err = vmp.UpdateParams()
	if err != nil {
		return vmp, err
	}

	return vmp, nil
}

// UpdateParams - reads configuration file and updates corresponding flags.
func (vmp *VictoriaMetricsParams) UpdateParams() error {
	err := vmp.loadVMAlertParams()
	if err != nil {
		return fmt.Errorf("cannot update VMAlertFlags config param: %w", err)
	}

	return nil
}

// loadVMAlertParams - load params and converts it to vmalert flags.
func (vmp *VictoriaMetricsParams) loadVMAlertParams() error {
	buf, err := os.ReadFile(vmp.BaseConfigPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("cannot read baseConfigPath for VMAlertParams: %w", err)
		}
		// fast return if users configuration doesn't exist with path
		// /srv/prometheus/prometheus.base.yml,
		// its maybe mounted into container by user.
		return nil
	}
	var cfg config.Config
	err = yaml.Unmarshal(buf, &cfg)
	if err != nil {
		return fmt.Errorf("cannot unmarshal baseConfigPath for VMAlertFlags: %w", err)
	}
	vmalertFlags := make([]string, 0, len(vmp.VMAlertFlags))
	for _, r := range cfg.RuleFiles {
		vmalertFlags = append(vmalertFlags, "--rule="+r)
	}
	if cfg.GlobalConfig.EvaluationInterval != 0 {
		vmalertFlags = append(vmalertFlags, "--evaluationInterval="+cfg.GlobalConfig.EvaluationInterval.String())
	}
	vmp.VMAlertFlags = vmalertFlags

	return nil
}

// ExternalVM returns true if VictoriaMetrics is configured to run externally.
func (vmp *VictoriaMetricsParams) ExternalVM() bool {
	return !internalAddr(vmp.url.Hostname())
}

// URL returns the base URL for VictoriaMetrics.
func (vmp *VictoriaMetricsParams) URL() string {
	return vmp.url.String()
}

// ParsedURL returns a copy of the parsed base URL for VictoriaMetrics. Callers may modify the copy.
func (vmp *VictoriaMetricsParams) ParsedURL() *url.URL {
	u := *vmp.url
	return &u
}

// URLFor returns the URL for a specific path in VictoriaMetrics.
func (vmp *VictoriaMetricsParams) URLFor(path string) (*url.URL, error) {
	if path == "" {
		return vmp.url, nil
	}
	return vmp.url.Parse(path)
}
