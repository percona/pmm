// pmm-managed
// Copyright (C) 2017 Percona LLC
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

package telemetry

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"

	guuid "github.com/google/uuid"
	"github.com/pkg/errors"
)

// for unit tests
var (
	stat     = os.Stat
	readFile = ioutil.ReadFile
	output   = commandOutput
)

func commandOutput(args ...string) ([]byte, error) {
	switch len(args) {
	case 0:
		return nil, fmt.Errorf("invalid number of arguments")
	case 1:
		return exec.Command(args[0]).Output()
	}
	return exec.Command(args[0], args[1:]...).Output()
}

func GenerateUUID() (string, error) {
	uuid, err := guuid.NewRandom()
	if err != nil {
		return "", errors.Wrap(err, "can't generate UUID")
	}

	// Old telemetry IDs have only 32 chars in the table but UUIDs + "-" = 36
	cleanUUID := strings.Replace(uuid.String(), "-", "", -1)
	return cleanUUID, nil
}

func getOSNameAndVersion() (string, error) {
	// freedesktop.org and systemd
	if _, err := stat("/etc/os-release"); err == nil {
		vals, err := getEntries("/etc/os-release", []string{"NAME", "VERSION"})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s", vals["NAME"], vals["VERSION"]), nil
	}

	// linuxbase.org
	if osName, err := output("lsb_release", "-si"); err == nil {
		if osVersion, err := output("lsb_release", "-sr"); err == nil {
			return fmt.Sprintf("%s %s", string(osName), string(osVersion)), nil
		}
		return "", errors.Wrap(err, "cannot get output of lsb_release -sr")
	}

	// For some versions of Debian/Ubuntu without lsb_release command
	if _, err := stat("/etc/lsb-release"); err == nil {
		vals, err := getEntries("/etc/lsb-release", []string{"DISTRIB_ID", "DISTRIB_RELEASE"})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s %s", vals["DISTRIB_ID"], vals["DISTRIB_RELEASE"]), nil
	}

	// Older Debian/Ubuntu/etc.
	if _, err := stat("/etc/debian_version"); err == nil {
		content, err := readFile("/etc/debian_version")
		if err != nil {
			return "", errors.Wrap(err, "cannot read /etc/debian_version")
		}
		return fmt.Sprintf("Debian %s", string(content)), nil
	}

	// Older Red Hat, CentOS, etc.
	if _, err := stat("/etc/redhat-release"); err == nil {
		content, err := readFile("/etc/redhat-release")
		if err != nil {
			return "", errors.Wrap(err, "cannot read /etc/redhat-release")
		}
		return string(content), nil
	}

	// Older SuSE
	if _, err := stat("/etc/SuSe-release"); err == nil {
		content, err := readFile("/etc/SuSe-release")
		if err != nil {
			return "", errors.Wrap(err, "cannot read /etc/SuSe-release")
		}
		return string(content), nil
	}

	// Fallback to generic os
	osName, err := output("uname", "-s")
	if err != nil {
		return "", errors.Wrap(err, "cannot get output of uname -s")
	}
	osVersion, err := output("uname", "-r")
	if err != nil {
		return "", errors.Wrap(err, "cannot get output of uname -r")
	}

	return fmt.Sprintf("%s %s", osName, osVersion), nil
}

func getEntries(filename string, keys []string) (map[string]string, error) {
	values := make(map[string]string)

	content, err := readFile(filename)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	re := regexp.MustCompile("^[\"'](.*)[\"']$")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		m := strings.Split(line, "=")
		if len(m) != 2 {
			continue
		}
		key := strings.ToLower(m[0])
		val := re.ReplaceAllString(m[1], "$1")
		for _, wantKey := range keys {
			if strings.ToLower(wantKey) == key {
				values[wantKey] = val
				continue
			}
		}
	}
	if len(values) < len(keys) {
		return nil, fmt.Errorf("Cannot get all entries %v from %s", keys, filename)
	}
	return values, nil
}
