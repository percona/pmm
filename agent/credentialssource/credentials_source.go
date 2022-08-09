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

// Package credentialssource provides managing of defaults file.
package credentialssource

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

// Parser is a struct which is responsible for parsing credentialsJson source/defaults file.
type Parser struct{}

// New creates new Parser.
func New() *Parser {
	return &Parser{}
}

type credentialsSource struct {
	username      string
	password      string
	host          string
	agentPassword string
	port          uint32
	socket        string
}

// credentialsJson provides access to an external provider so that
// the username, password, or agent password can be managed
// externally, e.g. HashiCorp Vault, Ansible Vault, etc.
type credentialsJson struct {
	AgentPassword string `json:"agentpassword"`
	Password      string `json:"password"`
	Username      string `json:"username"`
}

// ParseCredentialsSource parses given file in request. It returns the database specs.
func (d *Parser) ParseCredentialsSource(req *agentpb.ParseCredentialsSourceRequest) *agentpb.ParseCredentialsSourceResponse {
	var res agentpb.ParseCredentialsSourceResponse
	parsedData, err := parseCredentialsSourceFile(req.FilePath, req.ServiceType)
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

func parseCredentialsSourceFile(filePath string, serviceType inventorypb.ServiceType) (*credentialsSource, error) {
	if len(filePath) == 0 {
		return nil, errors.New("configPath for parseCredentialsSourceFile is empty")
	}

	filePath, err := expandPath(filePath)
	if err != nil {
		return nil, fmt.Errorf("fail to normalize path: %w", err)
	}

	// check if file exist
	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file doesn't exist: %s", filePath)
	}

	credentialsJsonFile, err := parseJsonFile(filePath)
	if err == nil {
		return credentialsJsonFile, nil
	}

	if serviceType == inventorypb.ServiceType_MYSQL_SERVICE {
		return parseMySQLDefaultsFile(filePath)
	}

	return nil, fmt.Errorf("unrecognized file type %s", filePath)
}

func parseJsonFile(filePath string) (*credentialsSource, error) {
	creds := credentialsJson{"", "", ""}

	f, err := os.Lstat(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if f.Mode()&0o111 != 0 {
		return nil, fmt.Errorf("%w: %s", errors.New("file execution is not supported"), filePath)
	}

	// Read the file
	content, err := readFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := json.Unmarshal([]byte(content), &creds); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	parsedData := &credentialsSource{
		username:      creds.Username,
		password:      creds.Password,
		agentPassword: creds.AgentPassword,
	}

	err = validateResults(parsedData)
	if err != nil {
		return nil, err
	}

	return parsedData, nil
}

func parseMySQLDefaultsFile(configPath string) (*credentialsSource, error) {
	cfg, err := ini.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("fail to read config file: %w", err)
	}

	cfgSection := cfg.Section("client")
	port, _ := cfgSection.Key("port").Uint()

	parsedData := &credentialsSource{
		username: cfgSection.Key("user").String(),
		password: cfgSection.Key("password").String(),
		host:     cfgSection.Key("host").String(),
		port:     uint32(port),
		socket:   cfgSection.Key("socket").String(),
	}

	err = validateResults(parsedData)
	if err != nil {
		return nil, err
	}

	return parsedData, nil
}

func validateResults(data *credentialsSource) error {
	if data.username == "" && data.password == "" && data.host == "" && data.port == 0 && data.socket == "" && data.agentPassword == "" {
		return errors.New("no data found in file")
	}
	return nil
}

func expandPath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		usr, err := user.Current()
		if err != nil {
			return "", fmt.Errorf("failed to expand path: %w", err)
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
