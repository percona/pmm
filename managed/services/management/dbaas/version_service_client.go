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

package dbaas

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/managed/utils/irt"
)

const (
	psmdbOperator = "psmdb-operator"
	pxcOperator   = "pxc-operator"
)

var errNoVersionsFound = errors.New("no versions to compare current version with found")

// componentVersion contains info about exact component version.
type componentVersion struct {
	ImagePath string `json:"imagePath"`
	ImageHash string `json:"imageHash"`
	Status    string `json:"status"`
	Critical  bool   `json:"critical"`
}

type matrix struct {
	Mongod        map[string]componentVersion `json:"mongod"`
	Pxc           map[string]componentVersion `json:"pxc"`
	Pmm           map[string]componentVersion `json:"pmm"`
	Proxysql      map[string]componentVersion `json:"proxysql"`
	Haproxy       map[string]componentVersion `json:"haproxy"`
	Backup        map[string]componentVersion `json:"backup"`
	Operator      map[string]componentVersion `json:"operator"`
	LogCollector  map[string]componentVersion `json:"logCollector"`
	PXCOperator   map[string]componentVersion `json:"pxcOperator,omitempty"`
	PSMDBOperator map[string]componentVersion `json:"psmdbOperator,omitempty"`
}

type Version struct {
	Product        string `json:"product"`
	ProductVersion string `json:"operator"`
	Matrix         matrix `json:"matrix"`
}

// VersionServiceResponse represents response from version service API.
type VersionServiceResponse struct {
	Versions []Version `json:"versions"`
}

// componentsParams contains params to filter components in version service API.
type componentsParams struct {
	product        string
	productVersion string
	dbVersion      string
}

// VersionServiceClient represents a client for Version Service API.
type VersionServiceClient struct {
	url  string
	http *http.Client
	irtm prom.Collector
	l    *logrus.Entry
}

// NewVersionServiceClient creates a new client for given version service URL.
func NewVersionServiceClient(url string) *VersionServiceClient {
	var t http.RoundTripper = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if logrus.GetLevel() >= logrus.TraceLevel {
		t = irt.WithLogger(t, logrus.WithField("component", "versionService/client").Tracef)
	}
	t, irtm := irt.WithMetrics(t, "versionService_client")

	return &VersionServiceClient{
		url: url,
		http: &http.Client{
			Transport: t,
		},
		irtm: irtm,
		l:    logrus.WithField("component", "VersionServiceClient"),
	}
}

// Describe implements prometheus.Collector.
func (c *VersionServiceClient) Describe(ch chan<- *prom.Desc) {
	c.irtm.Describe(ch)
}

// Collect implements prometheus.Collector.
func (c *VersionServiceClient) Collect(ch chan<- prom.Metric) {
	c.irtm.Collect(ch)
}

// Matrix calls version service with given params and returns components matrix.
func (c *VersionServiceClient) Matrix(ctx context.Context, params componentsParams) (*VersionServiceResponse, error) {
	paths := []string{c.url, params.product}
	if params.productVersion != "" {
		paths = append(paths, params.productVersion)
		if params.dbVersion != "" {
			paths = append(paths, params.dbVersion)
		}
	}
	url := strings.Join(paths, "/")
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vsResponse VersionServiceResponse
	err = json.Unmarshal(body, &vsResponse)
	if err != nil {
		return nil, err
	}

	return &vsResponse, nil
}

// IsDatabaseVersionSupportedByOperator returns false and err when request to version service fails. Otherwise returns boolean telling
// if given database version is supported by given operator version, error is nil in that case.
func (c *VersionServiceClient) IsDatabaseVersionSupportedByOperator(ctx context.Context, operatorType, operatorVersion, databaseVersion string) (bool, error) {
	m, err := c.Matrix(ctx, componentsParams{
		product:        operatorType,
		productVersion: operatorVersion,
		dbVersion:      databaseVersion,
	})
	if err != nil {
		return false, err
	}
	return len(m.Versions) != 0, nil
}

// IsOperatorVersionSupported returns true and nil if given operator version is supported in given PMM version.
// It returns false and error when fetching or parsing fails. False and nil when no error is encountered but
// version service does not have any matching versions.
func (c *VersionServiceClient) IsOperatorVersionSupported(ctx context.Context, operatorType string, pmmVersion string, operatorVersion string) (bool, error) {
	pmm, err := goversion.NewVersion(pmmVersion)
	if err != nil {
		return false, err
	}

	resp, err := c.Matrix(ctx, componentsParams{product: "pmm-server", productVersion: pmm.Core().String()})
	if err != nil {
		return false, err
	}

	if len(resp.Versions) == 0 {
		return false, nil
	}

	var operatorVersions map[string]componentVersion
	switch operatorType {
	case pxcOperator:
		operatorVersions = resp.Versions[0].Matrix.PXCOperator
	case psmdbOperator:
		operatorVersions = resp.Versions[0].Matrix.PSMDBOperator
	default:
		return false, errors.Errorf("%q is an unknown operator type", operatorType)
	}

	for version := range operatorVersions {
		if version == operatorVersion {
			return true, nil
		}
	}
	return false, nil
}

func latestRecommended(m map[string]componentVersion) (*goversion.Version, error) {
	if len(m) == 0 {
		return nil, errNoVersionsFound
	}
	latest := goversion.Must(goversion.NewVersion("0.0.0"))
	for version, component := range m {
		parsedVersion, err := goversion.NewVersion(version)
		if err != nil {
			return nil, err
		}
		if parsedVersion.GreaterThan(latest) && component.Status == "recommended" {
			latest = parsedVersion
		}
	}
	return latest, nil
}

// LatestOperatorVersion return latest recommended PXC and PSMDB operators for given PMM version.
func (c *VersionServiceClient) LatestOperatorVersion(ctx context.Context, pmmVersion string) (*goversion.Version, *goversion.Version, error) {
	if pmmVersion == "" {
		return nil, nil, errors.New("given PMM version is empty")
	}
	params := componentsParams{
		product:        "pmm-server",
		productVersion: pmmVersion,
	}
	resp, err := c.Matrix(ctx, params)
	if err != nil {
		return nil, nil, err
	}
	if len(resp.Versions) != 1 {
		return nil, nil, nil // no deps for the PMM version passed to c.Matrix
	}
	pmmVersionDeps := resp.Versions[0]
	latestPSMDBOperator, err := latestRecommended(pmmVersionDeps.Matrix.PSMDBOperator)
	if err != nil {
		return nil, nil, err
	}
	latestPXCOperator, err := latestRecommended(pmmVersionDeps.Matrix.PXCOperator)
	return latestPXCOperator, latestPSMDBOperator, err
}

// GetNextDatabaseImage returns image of the version that is a direct successor of currently installed version.
// It returns empty string if update is not available or error occurred.
func (c *VersionServiceClient) GetNextDatabaseImage(ctx context.Context, operatorType, operatorVersion, installedDBVersion string) (string, error) {
	// Get dependencies of operator type at given version.
	params := componentsParams{
		product:        operatorType,
		productVersion: operatorVersion,
	}
	matrix, err := c.Matrix(ctx, params)
	if err != nil {
		return "", err
	}
	if len(matrix.Versions) != 1 {
		return "", nil
	}
	operatorDependencies := matrix.Versions[0]

	// Choose proper versions map.
	var versions map[string]componentVersion
	switch operatorType {
	case psmdbOperator:
		versions = operatorDependencies.Matrix.Mongod
	case pxcOperator:
		versions = operatorDependencies.Matrix.Pxc
	default:
		return "", errors.Errorf("%q operator not supported", operatorType)
	}

	// Convert slice of version structs to slice of strings so it can be used in generic function next.
	stringVersions := make([]string, 0, len(versions))
	for version := range versions {
		stringVersions = append(stringVersions, version)
	}

	// Get direct successor of installed version.
	nextVersion, err := next(stringVersions, installedDBVersion)
	if err != nil {
		return "", err
	}
	if nextVersion == nil {
		return "", nil
	}
	return versions[nextVersion.String()].ImagePath, nil
}

// GetVersionServiceURL returns base URL for version service currently used
func (c *VersionServiceClient) GetVersionServiceURL() string {
	url, err := url.Parse(c.url)
	if err != nil {
		c.l.Warnf("failed to parse url %q: %v", c.url, err)
		return c.url
	}
	return url.Scheme + "://" + url.Host
}

// NextOperatorVersion returns operator version that is direct successor of currently installed one.
// It returns nil if update is not available or error occurred. It does not take PMM version into consideration.
// We need to upgrade to current + 1 version for upgrade to be successful. So even if dbaas-controller does not support the
// operator, we need to upgrade to it on our way to supported one.
func (c *VersionServiceClient) NextOperatorVersion(ctx context.Context, operatorType, installedVersion string) (nextOperatorVersion *goversion.Version, err error) {
	if installedVersion == "" {
		return
	}
	// Get all operator versions
	params := componentsParams{
		product: operatorType,
	}
	matrix, err := c.Matrix(ctx, params)
	if err != nil {
		return
	}
	if len(matrix.Versions) == 0 {
		return
	}

	// Convert slice of version structs to slice of strings so it can be used in generic function next.
	versions := make([]string, 0, len(matrix.Versions))
	for _, version := range matrix.Versions {
		versions = append(versions, version.ProductVersion)
	}

	// Find next versions if installed.
	if installedVersion != "" {
		return next(versions, installedVersion)
	}
	return
}

// next direct successor of given installed version, returns nil if there is none.
// An error is returned if any of given version can't be parsed. It's nil otherwise.
func next(versions []string, installedVersion string) (*goversion.Version, error) {
	if len(versions) == 0 {
		return nil, errNoVersionsFound
	}
	// Get versions greater than currently installed one.
	var nextVersion *goversion.Version
	installed, err := goversion.NewVersion(installedVersion)
	if err != nil {
		return nil, err
	}

	for _, version := range versions {
		v, err := goversion.NewVersion(version)
		if err != nil {
			return nil, err
		}
		if v.GreaterThan(installed) && (nextVersion == nil || nextVersion.GreaterThan(v)) {
			nextVersion = v
		}
	}

	return nextVersion, nil
}
