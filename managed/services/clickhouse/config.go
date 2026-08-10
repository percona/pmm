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

// Package clickhouse provides facilities for working with clickhouse.
package clickhouse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	defaultClickHouseConfig = "default"
	clickHouseConfigDir     = "/etc/clickhouse-server"
)

// stableConfigLinks are the fixed names ClickHouse is pointed at, paired with the suffix of the
// config-specific file each one must resolve to.
var stableConfigLinks = []struct{ link, suffix string }{
	{link: "config.xml", suffix: "-config.xml"},
	{link: "users.xml", suffix: "-users.xml"},
}

// GetClickHouseConfig returns the config name if the matching
// <config>-config.xml files exist on disk.
// Empty input falls back to defaultClickHouseConfig.
func GetClickHouseConfig(config string) (string, error) {
	if config == "" {
		return defaultClickHouseConfig, nil
	}

	return config, validateClickHouseConfigAt(config, clickHouseConfigDir)
}

// LinkClickHouseConfig points the fixed config.xml and users.xml names at the files of the
// given config.
//
// ClickHouse derives its merge directories from the paths it is handed rather than from the
// files those paths resolve to, so serving it the fixed names keeps drop-ins in
// /etc/clickhouse-server/config.d and /etc/clickhouse-server/users.d for every config.
//
// A path that already exists and is not a symlink is left untouched, since replacing it would
// discard a file somebody put there deliberately.
func LinkClickHouseConfig(config string) error {
	return linkClickHouseConfigAt(config, clickHouseConfigDir)
}

func linkClickHouseConfigAt(config, dir string) error {
	// Resolve every target before touching any link, so a missing file cannot leave the links
	// straddling two configs.
	targets := make([]string, len(stableConfigLinks))
	for i, l := range stableConfigLinks {
		target := filepath.Join(dir, config+l.suffix)
		_, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot stat %s: %w", target, err)
		}
		targets[i] = target
	}

	// Repointing only some of the links would leave ClickHouse reading a server config and a users
	// file from different configs, so a single unmanaged path opts all of them out.
	for _, l := range stableConfigLinks {
		link := filepath.Join(dir, l.link)

		fi, err := os.Lstat(link)
		switch {
		case err == nil && fi.Mode()&os.ModeSymlink == 0:
			logrus.WithField("path", link).Warn("ClickHouse config path is a regular file rather than a " +
				"symlink. Leaving all ClickHouse config links untouched, so PMM_CLICKHOUSE_CONFIG has no " +
				"effect. Remove the file to restore config switching.")
			return nil
		case err != nil && !errors.Is(err, os.ErrNotExist):
			return fmt.Errorf("cannot stat %s: %w", link, err)
		}
	}

	for i, l := range stableConfigLinks {
		link := filepath.Join(dir, l.link)
		target := targets[i]

		// Skip links that already resolve correctly. Beyond saving work, this keeps the common
		// case free of writes to /etc, which an arbitrary UID may not be allowed to perform.
		current, err := os.Readlink(link)
		if err == nil && current == target {
			continue
		}

		err = replaceSymlink(target, link)
		if err != nil {
			return fmt.Errorf("cannot point %s at %s: %w", link, target, err)
		}
	}

	return nil
}

// replaceSymlink points link at target, replacing link if it already exists.
func replaceSymlink(target, link string) error {
	tmp := link + ".tmp"

	err := os.Remove(tmp)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	err = os.Symlink(target, tmp)
	if err != nil {
		return err
	}

	err = os.Rename(tmp, link)
	if err != nil {
		return errors.Join(err, os.Remove(tmp))
	}

	return nil
}

// validateClickHouseConfigAt returns an error if configuration files are missing for given config.
func validateClickHouseConfigAt(config, dir string) error {
	// The config name is only ever a file name prefix. Anything else would let a path such as
	// ../../tmp/evil escape the config directory and point ClickHouse at an arbitrary file.
	if config != filepath.Base(config) || strings.HasPrefix(config, ".") {
		return fmt.Errorf("invalid PMM_CLICKHOUSE_CONFIG=%s: must be a name, not a path", config)
	}

	availableConfigs, err := availableClickHouseConfigs(dir)
	if err != nil {
		return fmt.Errorf("unable to get available ClickHouse configs: %w", err)
	}

	path := filepath.Join(dir, config+"-config.xml")
	_, err = os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf(
				"invalid PMM_CLICKHOUSE_CONFIG=%s: %s not found; available configs: %v",
				config, path, availableConfigs,
			)
		}
		return fmt.Errorf("cannot stat %s: %w", path, err)
	}

	return nil
}

// availableClickHouseConfigs lists config names that are present in the dir.
func availableClickHouseConfigs(dir string) ([]string, error) {
	var configs []string

	matches, err := filepath.Glob(filepath.Join(dir, "*-config.xml"))
	if err != nil {
		return nil, err
	}
	for _, m := range matches {
		name := strings.TrimSuffix(filepath.Base(m), "-config.xml")
		configs = append(configs, name)
	}

	sort.Strings(configs)
	return configs, nil
}
