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

package om

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/AlekSi/pointer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// om_bootstrap's release_match values: exact compares VERSION_ID whole, major its part before the first dot,
// as Rocky Linux 9.4 is release 9.
const (
	releaseMatchExact = "exact"
	releaseMatchMajor = "major"
)

// installSupport is om_bootstrap's table of the OS releases it installs each MongoDB version onto, or why PMM
// could not read it, which blocks every node.
type installSupport struct {
	distributions []extensionsSupportedOS
	err           error
}

// nodeRelease is a node's OS release as the table sees it.
type nodeRelease struct {
	osID      string
	versionID string
	// name is the release as the table names it, as "Rocky Linux 9.4", or by os_id for a distribution it lacks.
	name string
	// known reports whether the table has the node's distribution at all.
	known bool
	// versions are the major.minor MongoDB versions om_bootstrap installs onto the release.
	versions []string
}

func (s *Service) readInstallSupport(ctx context.Context) installSupport {
	if s.bootstrap == nil {
		return installSupport{err: status.Error(codes.FailedPrecondition, extensionsNotConfigured)}
	}
	distributions, err := s.bootstrap.supportedReleases(ctx)
	for _, distribution := range distributions {
		if distribution.ReleaseMatch != releaseMatchExact && distribution.ReleaseMatch != releaseMatchMajor {
			err = status.Errorf(codes.FailedPrecondition, "it matches %s releases by %s, which this PMM does not know",
				distribution.Name, distribution.ReleaseMatch)
		}
	}
	if err != nil {
		code := status.Code(err)
		// A missing route means an om_bootstrap app this PMM cannot work with, not a missing node.
		if code == codes.NotFound {
			code = codes.FailedPrecondition
		}
		return installSupport{err: status.Errorf(code,
			"PMM could not read from PMM Extensions which OS releases Operations installs onto: %s",
			status.Convert(err).Message())}
	}
	return installSupport{distributions: distributions}
}

func (s installSupport) release(observed map[string]any) nodeRelease {
	osID := pointer.GetString(observedString(observed, "os_id"))
	versionID := pointer.GetString(observedString(observed, "os_version_id"))
	release := nodeRelease{osID: osID, versionID: versionID, name: strings.TrimSpace(osID + " " + versionID)}
	i := slices.IndexFunc(s.distributions, func(distribution extensionsSupportedOS) bool {
		return distribution.OS == osID
	})
	if i < 0 {
		return release
	}
	distribution := s.distributions[i]
	release.known = true
	release.name = strings.TrimSpace(distribution.Name + " " + versionID)
	key := versionID
	if distribution.ReleaseMatch == releaseMatchMajor {
		key, _, _ = strings.Cut(versionID, ".")
	}
	for _, supported := range distribution.Releases {
		if versionID != "" && supported.Version == key {
			release.versions = supported.MongoDBVersions
		}
	}
	return release
}

// releaseReason says why Operations cannot install onto the node's OS release, or "" when it can, and whether
// that is a property of the node rather than a fault on it.
func (s installSupport) releaseReason(observed map[string]any) (string, bool) {
	release := s.release(observed)
	switch {
	case s.err != nil:
		return status.Convert(s.err).Message(), false
	case release.osID == "":
		return "no scan has reported this node's operating system yet", false
	case release.known && release.versionID == "":
		return "no scan has reported this node's OS release yet", false
	case len(release.versions) == 0:
		return fmt.Sprintf("this node runs %s, which Operations cannot install onto (supported: %s)",
			release.name, s.list()), true
	}
	return "", false
}

// list names every supported release, as "Ubuntu 22.04, 24.04; Rocky Linux 8, 9, 10".
func (s installSupport) list() string {
	groups := make([]string, 0, len(s.distributions))
	for _, distribution := range s.distributions {
		releases := make([]string, 0, len(distribution.Releases))
		for _, supported := range distribution.Releases {
			releases = append(releases, supported.Version)
		}
		groups = append(groups, distribution.Name+" "+strings.Join(releases, ", "))
	}
	return strings.Join(groups, "; ")
}

// refuseUnsupportedMongoDBVersion rejects a MongoDB version om_bootstrap does not install onto every node's release.
func (s installSupport) refuseUnsupportedMongoDBVersion(version string, nodeIDs []string, hosts []extensionsHost) error {
	wanted := majorMinor(version)
	blocked := make([]string, 0, len(hosts))
	for i, host := range hosts {
		release := s.release(host.Observed)
		if slices.Contains(release.versions, wanted) {
			continue
		}
		blocked = append(blocked, fmt.Sprintf("%s runs %s, which supports MongoDB %s",
			hostDisplayName(host, nodeIDs[i]), release.name, strings.Join(release.versions, ", ")))
	}
	if len(blocked) == 0 {
		return nil
	}
	return status.Errorf(codes.InvalidArgument,
		"MongoDB %s cannot be installed onto %d of the selected node(s) -- %s. Choose a version every selected node supports.",
		version, len(blocked), strings.Join(blocked, "; "))
}

// majorMinor cuts a MongoDB version to major.minor, which is what om_bootstrap's table lists.
func majorMinor(version string) string {
	major, rest, _ := strings.Cut(version, ".")
	minor, _, _ := strings.Cut(rest, ".")
	if minor == "" {
		return major
	}
	return major + "." + minor
}
