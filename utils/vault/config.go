// Copyright (C) 2023 Percona LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package vault reads secrets from HashiCorp Vault for pmm-agent and PMM Server.
package vault

import (
	"errors"
	"fmt"
	"net/url"
	"slices"
	"time"
)

// Supported authentication methods.
const (
	AuthMethodToken      = "token"
	AuthMethodAppRole    = "approle"
	AuthMethodKubernetes = "kubernetes"
	AuthMethodAWS        = "aws"
	AuthMethodAzure      = "azure"
	AuthMethodGCP        = "gcp"
)

// The types of the roles of the gcp method.
const (
	GCPRoleTypeGCE = "gce"
	GCPRoleTypeIAM = "iam"
)

// AuthMethods lists the supported authentication methods.
var AuthMethods = []string{ //nolint:gochecknoglobals
	AuthMethodToken, AuthMethodAppRole, AuthMethodKubernetes, AuthMethodAWS, AuthMethodAzure, AuthMethodGCP,
}

const (
	// DefaultCacheTTL is how long a secret without a lease is reused before it is read again.
	DefaultCacheTTL = time.Minute

	// DefaultKubernetesJWTFile is where Kubernetes mounts the service account token.
	DefaultKubernetesJWTFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"

	redacted = "***"
)

// Config configures access to Vault.
type Config struct {
	// Address is the Vault URL, e.g. https://vault.example.com:8200. Vault is not used when it is empty.
	Address       string `yaml:"address,omitempty"`
	Namespace     string `yaml:"namespace,omitempty"`
	CACert        string `yaml:"ca-cert,omitempty"`
	TLSSkipVerify bool   `yaml:"tls-skip-verify,omitempty"`

	AuthMethod string `yaml:"auth-method,omitempty"`
	// AuthMount is the path the auth method is mounted at; it defaults to the method name.
	AuthMount string `yaml:"auth-mount,omitempty"`

	// Token and TokenFile are used by the token method. TokenFile is read again whenever Vault rejects
	// the token, so it works with a Vault Agent sink.
	Token     string `yaml:"token,omitempty"`
	TokenFile string `yaml:"token-file,omitempty"`

	// RoleID, SecretID and SecretIDFile are used by the approle method.
	RoleID       string `yaml:"role-id,omitempty"`
	SecretID     string `yaml:"secret-id,omitempty"`
	SecretIDFile string `yaml:"secret-id-file,omitempty"`

	// Role is the Vault role of the kubernetes, aws, azure and gcp methods.
	Role string `yaml:"role,omitempty"`
	// JWTFile is the service account token of the kubernetes method.
	JWTFile string `yaml:"jwt-file,omitempty"`
	// AWSRegion selects the regional STS endpoint of the aws method; the global one is used when empty.
	AWSRegion string `yaml:"aws-region,omitempty"`
	// AWSServerID is sent as X-Vault-AWS-IAM-Server-ID when the aws method requires it.
	AWSServerID string `yaml:"aws-server-id,omitempty"`
	// AzureResource is the resource that the azure method requests the access token of the managed identity for,
	// DefaultAzureResource when empty.
	AzureResource string `yaml:"azure-resource,omitempty"`
	// AzureClientID selects a user-assigned managed identity for the azure method; the system-assigned one is used
	// when empty.
	AzureClientID string `yaml:"azure-client-id,omitempty"`
	// GCPRoleType is the type of the role of the gcp method: gce, the default, or iam.
	GCPRoleType string `yaml:"gcp-role-type,omitempty"`

	// CacheTTL is how long a secret without a lease is reused before it is read again.
	CacheTTL time.Duration `yaml:"cache-ttl,omitempty"`
}

// String hides the credentials of the configuration, so that it can be logged.
func (c Config) String() string {
	type plain Config
	if c.Token != "" {
		c.Token = redacted
	}
	if c.SecretID != "" {
		c.SecretID = redacted
	}
	return fmt.Sprintf("%+v", plain(c))
}

// Enabled reports whether Vault is configured.
func (c Config) Enabled() bool {
	return c.Address != ""
}

// Validate returns an error if an enabled configuration is incomplete.
func (c Config) Validate() error {
	if !c.Enabled() {
		return nil
	}

	u, err := url.Parse(c.Address)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("vault address must be an http(s) URL, got '%s'", c.Address)
	}

	if !slices.Contains(AuthMethods, c.authMethod()) {
		return fmt.Errorf("unsupported vault auth method '%s', expected one of %v", c.AuthMethod, AuthMethods)
	}

	switch c.authMethod() {
	case AuthMethodToken:
		if c.Token == "" && c.TokenFile == "" {
			return errors.New("vault auth method token requires a token or a token file")
		}
	case AuthMethodAppRole:
		if c.RoleID == "" || (c.SecretID == "" && c.SecretIDFile == "") {
			return errors.New("vault auth method approle requires a role ID and a secret ID or a secret ID file")
		}
	case AuthMethodKubernetes, AuthMethodAWS, AuthMethodAzure, AuthMethodGCP:
		if c.Role == "" {
			return fmt.Errorf("vault auth method %s requires a role", c.authMethod())
		}
	}

	if c.authMethod() == AuthMethodGCP && !slices.Contains([]string{"", GCPRoleTypeGCE, GCPRoleTypeIAM}, c.GCPRoleType) {
		return fmt.Errorf("unsupported vault gcp role type '%s', expected %s or %s", c.GCPRoleType, GCPRoleTypeGCE, GCPRoleTypeIAM)
	}

	if c.CacheTTL < 0 {
		return errors.New("vault cache TTL must not be negative")
	}

	return nil
}

func (c Config) authMethod() string {
	if c.AuthMethod == "" {
		return AuthMethodToken
	}
	return c.AuthMethod
}

func (c Config) authMount() string {
	if c.AuthMount == "" {
		return c.authMethod()
	}
	return c.AuthMount
}

// EffectiveCacheTTL returns CacheTTL, or its default when it is not set.
func (c Config) EffectiveCacheTTL() time.Duration {
	if c.CacheTTL == 0 {
		return DefaultCacheTTL
	}
	return c.CacheTTL
}
