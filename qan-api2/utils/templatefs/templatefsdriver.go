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

package templatefs

import (
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/golang-migrate/migrate/v4/source"
)

type Driver struct {
	fs    *TemplateFS
	files []fs.DirEntry
	dir   string
}

func NewDriver(fs *TemplateFS, dir string) (*Driver, error) {
	files, err := fs.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	return &Driver{fs: fs, files: files, dir: dir}, nil
}

func (d *Driver) Open(url string) (source.Driver, error) {
	return d, nil
}

func (d *Driver) Close() error {
	return nil
}

func (d *Driver) First() (version uint, err error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, err := parseVersion(f.Name())
			return v, err
		}
	}
	return 0, io.EOF
}

func (d *Driver) Prev(version uint) (prevVersion uint, err error) {
	var last uint
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if v < version && v > last {
				last = v
			}
		}
	}
	if last == 0 {
		return 0, io.EOF
	}
	return last, nil
}

func (d *Driver) Next(version uint) (nextVersion uint, err error) {
	found := false
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if found {
				return v, nil
			}
			if v == version {
				found = true
			}
		}
	}
	return 0, io.EOF
}

func (d *Driver) ReadUp(version uint) (r io.ReadCloser, identifier string, err error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".up.sql") {
			v, _ := parseVersion(f.Name())
			if v == version {
				b, err := d.fs.ReadFile(path.Join(d.dir, f.Name()))
				if err != nil {
					return nil, "", err
				}
				return io.NopCloser(strings.NewReader(string(b))), f.Name(), nil
			}
		}
	}
	return nil, "", io.EOF
}

func (d *Driver) ReadDown(version uint) (r io.ReadCloser, identifier string, err error) {
	for _, f := range d.files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".down.sql") {
			v, _ := parseVersion(f.Name())
			if v == version {
				b, err := d.fs.ReadFile(path.Join(d.dir, f.Name()))
				if err != nil {
					return nil, "", err
				}
				return io.NopCloser(strings.NewReader(string(b))), f.Name(), nil
			}
		}
	}
	return nil, "", io.EOF
}

func (d *Driver) SetVersion(version uint, dirty bool) error {
	return nil
}

func (d *Driver) Version() (version uint, dirty bool, err error) {
	return 0, false, nil
}

func (d *Driver) Drop() error {
	return nil
}

func parseVersion(name string) (uint, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid migration filename: %s", name)
	}
	var v uint
	_, err := fmt.Sscanf(parts[0], "%d", &v)
	return v, err
}
