// Copyright 2019 Percona LLC
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

// Package serviceparamssource provides managing of service parameters source file.
package serviceparamssource

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

// Parser is a struct which is responsible for parsing service parameters file json source/mysql defaults file.
type Parser struct{}

// New creates new Parser.
func New() *Parser {
	return &Parser{}
}

type serviceParamsSource struct {
	username      string
	password      string
	host          string
	agentPassword string
	port          uint32
	socket        string
}

// serviceParamsSourceJSON provides access to an external provider so that
// the username, password, or agent password can be managed
// externally, e.g. HashiCorp Vault, Ansible Vault, etc.
type serviceParamsSourceJSON struct {
	AgentPassword string `json:"agentpassword"`
	Password      string `json:"password"`
	Username      string `json:"username"`
}

// ParseServiceParamsSource parses given file in request. It returns the database specs.
func (d *Parser) ParseServiceParamsSource(req *agentpb.ParseServiceParamsSourceRequest) *agentpb.ParseServiceParamsSourceResponse {
	var res agentpb.ParseServiceParamsSourceResponse
	parsedData, err := parseServiceParamsSourceFile(req.FilePath, req.ServiceType)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	err = validateResults(parsedData)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	res.Username = parsedData.username
	res.Password = parsedData.password
	res.AgentPassword = parsedData.agentPassword
	res.Host = parsedData.host
	res.Port = parsedData.port
	res.Socket = parsedData.socket

	return &res
}

func parseServiceParamsSourceFile(filePath string, serviceType inventorypb.ServiceType) (*serviceParamsSource, error) {
	if filePath == "" {
		return nil, errors.New("configPath for parseServiceParamsSourceFile is empty")
	}

	filePath, err := expandPath(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to normalize path: %s", filePath)
	}

	// check if file exist
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil, errors.Errorf("file doesn't exist: %s", filePath)
	}

	parametersJSONFile, err := parseJSONFile(filePath)
	if err == nil {
		return parametersJSONFile, nil
	}

	if serviceType == inventorypb.ServiceType_MYSQL_SERVICE {
		return parseMySQLDefaultsFile(filePath)
	}

	return nil, errors.Wrapf(err, "unrecognized file type %s", filePath)
}

func parseJSONFile(filePath string) (*serviceParamsSource, error) {
	// Read the file
	content, err := readFile(filePath)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot read file %s", filePath)
	}

	var parameters serviceParamsSourceJSON
	if err := json.Unmarshal([]byte(content), &parameters); err != nil {
		return nil, errors.Wrapf(err, "cannot umarshal file %s", filePath)
	}

	parsedData := &serviceParamsSource{
		username:      parameters.Username,
		password:      parameters.Password,
		agentPassword: parameters.AgentPassword,
	}

	return parsedData, nil
}

func parseMySQLDefaultsFile(configPath string) (*serviceParamsSource, error) {
	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to read config file: %s", configPath)
	}

	cfgSection := cfg.Section("client")
	port, _ := cfgSection.Key("port").Uint()

	parsedData := &serviceParamsSource{
		username: cfgSection.Key("user").String(),
		password: cfgSection.Key("password").String(),
		host:     cfgSection.Key("host").String(),
		port:     uint32(port),
		socket:   cfgSection.Key("socket").String(),
	}

	return parsedData, nil
}

func validateResults(data *serviceParamsSource) error {
	if data.username == "" && data.password == "" && data.host == "" && data.port == 0 && data.socket == "" && data.agentPassword == "" {
		return errors.New("no data found in file")
	}
	return nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", errors.Wrapf(err, "failed to expand path: %s", path)
		}
		return filepath.Join(usr.HomeDir, path[2:]), nil
	}
	return path, nil
}

// readFile reads file from filepath if filepath is not empty.
func readFile(filePath string) (string, error) {
	if filePath == "" {
		return "", nil
	}

	content, err := os.ReadFile(filepath.Clean(filePath))
	if err != nil {
		return "", errors.Wrapf(err, "cannot load file in path %q", filePath)
	}

	return string(content), nil
}
