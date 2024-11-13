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

package checks

import (
	"net"
	"strconv"

	"github.com/percona/saas/pkg/starlark"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/percona/pmm/version"
)

var privateNetworks []*net.IPNet

// GetFuncsForVersion returns predefined functions for specified check version.
func GetFuncsForVersion(version uint32) (map[string]starlark.GoFunc, error) {
	switch version {
	case 1, 2:
		return map[string]starlark.GoFunc{
			"parse_version":      parseVersion,
			"format_version_num": formatVersionNum,
		}, nil
	default:
		return nil, errors.Errorf("unsupported check version: %d", version)
	}
}

// parseVersion accepts a single string argument (version), and returns map[string]interface{}
// with keys: major, minor, patch (int64), num (MMmmpp, int64), and rest (string).
func parseVersion(args ...interface{}) (interface{}, error) {
	if l := len(args); l != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", l)
	}

	s, ok := args[0].(string)
	if !ok {
		return nil, errors.Errorf("expected string argument, got %[1]T (%[1]v)", args[0])
	}

	p, err := version.Parse(s)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"major":   int64(p.Major),
		"minor":   int64(p.Minor),
		"patch":   int64(p.Patch),
		"rest":    p.Rest,
		"num":     int64(p.Num),
		"numrest": int64(p.NumRest),
	}, nil
}

// formatVersionNum accepts a single int64 argument (version num MMmmpp or MMmmppRRR), and returns
// MM.mm.pp or MM.mm.pp-RRR as a string.
func formatVersionNum(args ...interface{}) (interface{}, error) {
	if l := len(args); l != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", l)
	}

	num, ok := args[0].(int64)
	if !ok {
		return nil, errors.Errorf("expected int64 argument, got %[1]T (%[1]v)", args[0])
	}
	// process numbers with a rest part included
	if num > 10000000 {
		p := &version.Parsed{
			Major:   int(num / 10000000),
			Minor:   int(num / 100000 % 100),
			Patch:   int(num / 1000 % 100),
			Rest:    "-" + strconv.FormatInt(num%1000, 10),
			NumRest: int(num % 1000),
		}

		return p.String(), nil
	}

	p := &version.Parsed{
		Major: int(num / 10000),
		Minor: int(num / 100 % 100),
		Patch: int(num % 100),
	}

	return p.String(), nil
}

// GetAdditionalContext returns additional functions to be used in check scripts.
func GetAdditionalContext() map[string]starlark.GoFunc {
	return map[string]starlark.GoFunc{
		"ip_is_private":      ipIsPrivate,
		"parse_version":      parseVersion,
		"format_version_num": formatVersionNum,
	}
}

// ipIsPrivate accepts a single string argument (IP address or a network) and
// returns true for a private address, otherwise false. It returns nil in case of an invalid argument.
func ipIsPrivate(args ...interface{}) (interface{}, error) {
	log := logrus.WithField("component", "checks")

	if l := len(args); l != 1 {
		return nil, errors.Errorf("expected 1 argument, got %d", l)
	}

	ip, ok := args[0].(string)
	if !ok {
		return nil, errors.Errorf("expected string argument, got %[1]T (%[1]v)", args[0])
	}

	ipAddress := net.ParseIP(ip)
	if ipAddress == nil {
		// check if string was in CIDR notation
		_, net, err := net.ParseCIDR(ip)
		if err != nil {
			log.Errorf("invalid ip/network address: %q", ip)
			return nil, nil //nolint:nilnil,nilerr
		}
		for _, network := range privateNetworks {
			// check if the two networks intersect
			// https://stackoverflow.com/questions/34729158/how-to-detect-if-two-golang-net-ipnet-objects-intersect/34729915#34729915
			if net.Contains(network.IP) || network.Contains(net.IP) {
				return true, nil
			}
		}
		return false, nil
	}

	for _, network := range privateNetworks {
		if network.Contains(ipAddress) {
			return true, nil
		}
	}
	return false, nil
}

//nolint:gochecknoinits
func init() {
	// full list of reserved network addresses https://en.wikipedia.org/wiki/Reserved_IP_addresses
	privateAddressBlocks := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"198.18.0.0/15",
		"169.254.0.0/16",
		"224.0.0.0/24",
		"127.0.0.0/8",

		"fc00::/7",
		"fe80::/10",
		"::1/128",
	}

	for _, b := range privateAddressBlocks {
		_, network, err := net.ParseCIDR(b)
		if err != nil {
			panic(err)
		}
		privateNetworks = append(privateNetworks, network)
	}
}
