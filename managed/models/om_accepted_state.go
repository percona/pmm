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

package models

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AcceptedStateSchemaVersion is the document version this code writes; it lives in the
// om_cluster_states.schema_version column, not in the document.
const AcceptedStateSchemaVersion int32 = 1

// AcceptedState is a cluster's desired MongoDB and process configuration, keyed by our own ids
// (stable across PMM re-registration) and holding no node/service/agent ids or live readings.
type AcceptedState struct {
	FCV         string                        `json:"fcv"`
	Auth        AcceptedAuth                  `json:"auth"`
	ReplicaSets map[string]AcceptedReplicaSet `json:"replica_sets"`
	Members     map[string]AcceptedMember     `json:"members"`
	Backup      AcceptedBackup                `json:"backup"`
}

// AcceptedAuth is the cluster's authentication configuration. TLS certificate material is held as
// om_secrets references, never inline.
type AcceptedAuth struct {
	Mechanisms []string    `json:"mechanisms"`
	Membership string      `json:"membership"`
	TLS        AcceptedTLS `json:"tls"`
}

// AcceptedTLS carries the cluster's TLS mode and certificate references.
type AcceptedTLS struct {
	Mode      string `json:"mode,omitempty"`
	CARef     string `json:"ca_ref,omitempty"`
	ClientRef string `json:"client_ref,omitempty"`
}

// AcceptedReplicaSet is one replica set's desired settings; its ObservedFrom is kept for reference,
// not drift-compared.
type AcceptedReplicaSet struct {
	Settings     map[string]any       `json:"settings,omitempty"`
	ObservedFrom AcceptedObservedFrom `json:"observed_from"`
}

// AcceptedObservedFrom records the replica-set config a state was observed from.
type AcceptedObservedFrom struct {
	ConfigVersion int32 `json:"config_version"`
	Term          int64 `json:"term"`
}

// AcceptedControl is how a member's process is controlled.
type AcceptedControl struct {
	Method string `json:"method"`
	Unit   string `json:"unit,omitempty"`
}

// AcceptedInstall is how a member's binary was installed.
type AcceptedInstall struct {
	Method   string   `json:"method"`
	Repo     string   `json:"repo,omitempty"`
	Packages []string `json:"packages,omitempty"`
}

// AcceptedConfig is a member's config file and its parsed form; parsed secret leaves are
// om_secrets references, never inline.
type AcceptedConfig struct {
	File   string         `json:"file,omitempty"`
	Parsed map[string]any `json:"parsed,omitempty"`
}

// AcceptedMember is one member's desired state; a mongos carries only the process-level fields and
// none of the replica-set-only fields.
type AcceptedMember struct {
	Process        string            `json:"process"`
	Host           string            `json:"host"`
	ReplicaSet     string            `json:"replica_set,omitempty"`
	BinaryVersion  string            `json:"binary_version"`
	Control        AcceptedControl   `json:"control"`
	Install        AcceptedInstall   `json:"install"`
	Config         AcceptedConfig    `json:"config"`
	ConfigSHA256   string            `json:"config_sha256"`
	RSMemberID     *int32            `json:"rs_member_id,omitempty"`
	Votes          *int32            `json:"votes,omitempty"`
	Priority       *float64          `json:"priority,omitempty"`
	Hidden         *bool             `json:"hidden,omitempty"`
	SecondaryDelay *int64            `json:"secondary_delay_secs,omitempty"`
	ArbiterOnly    *bool             `json:"arbiter_only,omitempty"`
	BuildIndexes   *bool             `json:"build_indexes,omitempty"`
	Tags           map[string]string `json:"tags,omitempty"`
	DataPath       string            `json:"data_path,omitempty"`
	KeyfilePath    string            `json:"keyfile_path,omitempty"`
	LogPath        string            `json:"log_path,omitempty"`
}

// AcceptedBackup is the cluster-wide backup ownership (D13): "operations", or unset.
type AcceptedBackup struct {
	Owner string `json:"owner,omitempty"`
}

// Member processes within the accepted-state document.
const (
	acceptedProcessMongod = "mongod"
	acceptedProcessMongos = "mongos"
)

// Validate checks the closed field list: required fields are present, every member names a known
// process, and a mongos carries none of the replica-set-only fields.
func (s *AcceptedState) Validate() error {
	if s.FCV == "" {
		return errors.New("accepted state: fcv is required")
	}
	if len(s.Auth.Mechanisms) == 0 {
		return errors.New("accepted state: auth.mechanisms is required")
	}
	if len(s.ReplicaSets) == 0 {
		return errors.New("accepted state: at least one replica set is required")
	}
	if len(s.Members) == 0 {
		return errors.New("accepted state: at least one member is required")
	}
	for id, m := range s.Members {
		err := m.validate(id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *AcceptedMember) validate(id string) error {
	if m.Process != acceptedProcessMongod && m.Process != acceptedProcessMongos {
		return fmt.Errorf("accepted state: member %s has unknown process '%s'", id, m.Process)
	}
	if m.Host == "" {
		return fmt.Errorf("accepted state: member %s is missing host", id)
	}
	if m.BinaryVersion == "" {
		return fmt.Errorf("accepted state: member %s is missing binary_version", id)
	}
	if m.Control.Method == "" {
		return fmt.Errorf("accepted state: member %s is missing control.method", id)
	}
	if m.Install.Method == "" {
		return fmt.Errorf("accepted state: member %s is missing install.method", id)
	}
	if m.ConfigSHA256 == "" {
		return fmt.Errorf("accepted state: member %s is missing config_sha256", id)
	}
	if m.Process == acceptedProcessMongos {
		return m.validateMongos(id)
	}
	if m.RSMemberID == nil {
		return fmt.Errorf("accepted state: mongod member %s is missing rs_member_id", id)
	}
	return nil
}

// validateMongos rejects any replica-set-only field on a mongos member.
func (m *AcceptedMember) validateMongos(id string) error {
	switch {
	case m.RSMemberID != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set rs_member_id", id)
	case m.Votes != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set votes", id)
	case m.Priority != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set priority", id)
	case m.Hidden != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set hidden", id)
	case m.SecondaryDelay != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set secondary_delay_secs", id)
	case m.ArbiterOnly != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set arbiter_only", id)
	case m.BuildIndexes != nil:
		return fmt.Errorf("accepted state: mongos member %s must not set build_indexes", id)
	case len(m.Tags) != 0:
		return fmt.Errorf("accepted state: mongos member %s must not set tags", id)
	case m.DataPath != "" || m.KeyfilePath != "" || m.LogPath != "":
		return fmt.Errorf("accepted state: mongos member %s must not set data/keyfile/log paths", id)
	}
	return nil
}

// MarshalAcceptedState validates and serialises a document for om_cluster_states.document.
func MarshalAcceptedState(s *AcceptedState) ([]byte, error) {
	err := s.Validate()
	if err != nil {
		return nil, err
	}
	return json.Marshal(s)
}

// UnmarshalAcceptedState parses and validates a document. Unknown top-level fields are tolerated.
func UnmarshalAcceptedState(document []byte) (*AcceptedState, error) {
	var s AcceptedState
	err := json.Unmarshal(document, &s)
	if err != nil {
		return nil, fmt.Errorf("accepted state: %w", err)
	}
	err = s.Validate()
	if err != nil {
		return nil, err
	}
	return &s, nil
}
