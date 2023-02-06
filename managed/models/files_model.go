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

package models

import (
	"fmt"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
)

//go:generate ../../bin/reform

// File represents File as stored in database.
//
//reform:files
type File struct {
	Name      string    `reform:"name,pk"`
	Content   []byte    `reform:"content"`
	UpdatedAt time.Time `reform:"updated_at"`
}

// ErrFileNotFound is returned when a file is not found.
var ErrFileNotFound = fmt.Errorf("File not found")

// InsertFileParams represent insert file params. Call validate before use.
type InsertFileParams struct {
	Name    string
	Content []byte
}

func checkFileName(name string) error {
	switch want := govalidator.SafeFileName(name); {
	case name == ".", name == "..":
		fallthrough
	case len(want) == 0:
		return fmt.Errorf(`invalid file name: %s`, name)
	case name != want:
		return fmt.Errorf(`invalid file name: %s; want: %s`, name, want)
	}
	return nil
}

// Validate validates params.
func (p *InsertFileParams) Validate() error {
	if err := checkFileName(p.Name); err != nil {
		return err
	}
	return nil
}

// UpdateFileParams represent update file params.
type UpdateFileParams struct {
	Name    string
	Content []byte
}

// Validate validates params.
func (p *UpdateFileParams) Validate() error {
	if len(p.Name) == 0 {
		return errors.New("empty name for file to update")
	}
	return nil
}
