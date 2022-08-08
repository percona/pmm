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
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"

	"github.com/percona/pmm/api/agentpb"
	"github.com/percona/pmm/api/inventorypb"
)

// Parser is a struct which is responsible for parsing credentials source/defaults file.
type Parser struct{}

// New creates new Parser.
func New() *Parser {
	return &Parser{}
}

type credentialsSource struct {
	username      string
	password      string
	host          string
	agetnPassword string
	port          uint32
	socket        string
}

// credentials provides access to an external provider so that
// the username, password, or agent password can be managed
// externally, e.g. HashiCorp Vault, Ansible Vault, etc.
type credentials struct {
	AgentPassword string `json:"agentpassword"`
	Password      string `json:"password"`
	Username      string `json:"username"`
}

// ParseCredentialsSource parses given file in request. It returns the database specs.
func (d *Parser) ParseCredentialsSource(req *agentpb.ParseCredentialsSourceRequest) *agentpb.ParseCredentialsSourceResponse {
	var res agentpb.ParseCredentialsSourceResponse
	creds, err := parseCredentialsSourceFile(req.FilePath, req.ServiceType)
	if err != nil {
		res.Error = err.Error()
		return &res
	}

	res.Username = creds.username
	res.Password = creds.password
	res.Host = creds.host
	res.Port = creds.port
	res.Socket = creds.socket

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

	// open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	contentType, err := getFileContentType(file)
	if err != nil {
		return nil, err
	}

	switch contentType {
	case "application/json":
		return parseJsonFile(filePath)
	case "application/ini":
		return parseIniFile(filePath, serviceType)
	default:
		return nil, errors.Errorf("unsupported file type %s", contentType)
	}
}

func parseJsonFile(filePath string) (*credentialsSource, error) {
	creds, err := readCredentialsFromSource(filePath)
	if err != nil {
		return nil, err
	}

	return &credentialsSource{
		username:      creds.Username,
		password:      creds.Password,
		agetnPassword: creds.AgentPassword,
	}, nil
}

func parseIniFile(filePath string, serviceType inventorypb.ServiceType) (*credentialsSource, error) {
	switch serviceType {
	case inventorypb.ServiceType_MYSQL_SERVICE:
		return parseMySQLDefaultsFile(filePath)
	case inventorypb.ServiceType_EXTERNAL_SERVICE:
	case inventorypb.ServiceType_HAPROXY_SERVICE:
	case inventorypb.ServiceType_MONGODB_SERVICE:
	case inventorypb.ServiceType_POSTGRESQL_SERVICE:
	case inventorypb.ServiceType_PROXYSQL_SERVICE:
	case inventorypb.ServiceType_SERVICE_TYPE_INVALID:
		return nil, errors.Errorf("unimplemented service type %s", serviceType)
	}

	return nil, errors.Errorf("unimplemented service type %s", serviceType)
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

	err = validateDefaultsFileResults(parsedData)
	if err != nil {
		return nil, err
	}

	return parsedData, nil
}

func validateDefaultsFileResults(data *credentialsSource) error {
	if data.username == "" && data.password == "" && data.host == "" && data.port == 0 && data.socket == "" {
		return errors.New("no data found in defaults file")
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

// readCredentialsFromSource parses a JSON file src and return
// a credentials pointer containing the data.
func readCredentialsFromSource(src string) (*credentials, error) {
	creds := credentials{"", "", ""}

	f, err := os.Lstat(src)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if f.Mode()&0o111 != 0 {
		return nil, fmt.Errorf("%w: %s", errors.New("execution is not supported"), src)
	}

	// Read the file
	content, err := readFile(src)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	if err := json.Unmarshal([]byte(content), &creds); err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &creds, nil
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

//
func getFileContentType(file *os.File) (string, error) {

	buf := make([]byte, 512)

	_, err := file.Read(buf)

	if err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buf)

	return contentType, nil
}
