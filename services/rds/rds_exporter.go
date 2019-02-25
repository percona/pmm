// pmm-managed
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

package rds

/*
import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type instanceType string

const (
	unknown     instanceType = "unknown"
	auroraMySQL instanceType = "aurora_mysql"
	mySQL       instanceType = "mysql"
)

type rdsExporterInstance struct {
	Region       string       `yaml:"region"`
	Instance     string       `yaml:"instance"`
	Type         instanceType `yaml:"type"`
	AWSAccessKey *string      `yaml:"aws_access_key,omitempty"`
	AWSSecretKey *string      `yaml:"aws_secret_key,omitempty"`
}

type rdsExporterConfig struct {
	Instances []rdsExporterInstance `yaml:"instances"`
}

func (config *rdsExporterConfig) Marshal() ([]byte, error) {
	b, err := yaml.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "can't marshal rds_exporter configuration file")
	}
	b = append([]byte("# Managed by pmm-managed. DO NOT EDIT.\n---\n"), b...)
	return b, nil
}
*/
