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

package models_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/percona/pmm/managed/models"
)

const testRSID = "11111111-1111-1111-1111-111111111111"

func validMongod(rs string, memberID int32) models.AcceptedMember {
	return models.AcceptedMember{
		Process:       "mongod",
		Host:          "node-a:27017",
		ReplicaSet:    rs,
		BinaryVersion: "7.0.14",
		Control:       models.AcceptedControl{Method: "systemd", Unit: "mongod.service"},
		Install:       models.AcceptedInstall{Method: "package", Repo: "percona"},
		Config:        models.AcceptedConfig{File: "/etc/mongod.conf"},
		ConfigSHA256:  "0000000000000000000000000000000000000000000000000000000000000000",
		RSMemberID:    new(memberID),
	}
}

func validMongos() models.AcceptedMember {
	return models.AcceptedMember{
		Process:       "mongos",
		Host:          "router-a:27017",
		BinaryVersion: "7.0.14",
		Control:       models.AcceptedControl{Method: "systemd", Unit: "mongos.service"},
		Install:       models.AcceptedInstall{Method: "package"},
		Config:        models.AcceptedConfig{File: "/etc/mongos.conf"},
		ConfigSHA256:  "1111111111111111111111111111111111111111111111111111111111111111",
	}
}

func validAcceptedState() *models.AcceptedState {
	return &models.AcceptedState{
		FCV:         "7.0",
		Auth:        models.AcceptedAuth{Mechanisms: []string{"SCRAM-SHA-256"}, Membership: "keyfile"},
		ReplicaSets: map[string]models.AcceptedReplicaSet{testRSID: {ObservedFrom: models.AcceptedObservedFrom{ConfigVersion: 1, Term: 1}}},
		Members:     map[string]models.AcceptedMember{"aaaaaaaa-0000-0000-0000-000000000001": validMongod(testRSID, 0)},
		Backup:      models.AcceptedBackup{Owner: "operations"},
	}
}

func TestAcceptedStateValid(t *testing.T) {
	t.Run("round-trips a valid document", func(t *testing.T) {
		doc, err := models.MarshalAcceptedState(validAcceptedState())
		require.NoError(t, err)
		back, err := models.UnmarshalAcceptedState(doc)
		require.NoError(t, err)
		assert.Equal(t, validAcceptedState(), back)
	})

	t.Run("accepts a mongos carrying only process-level fields", func(t *testing.T) {
		s := validAcceptedState()
		s.Members["bbbbbbbb-0000-0000-0000-000000000001"] = validMongos()
		_, err := models.MarshalAcceptedState(s)
		require.NoError(t, err)
	})

	t.Run("tolerates an unknown additive top-level section", func(t *testing.T) {
		doc, err := models.MarshalAcceptedState(validAcceptedState())
		require.NoError(t, err)
		extended := string(doc[:len(doc)-1]) + `,"pitr":{"enabled":true}}`
		_, err = models.UnmarshalAcceptedState([]byte(extended))
		require.NoError(t, err)
	})
}

func TestAcceptedStateInvalid(t *testing.T) {
	cases := map[string]func(*models.AcceptedState){
		"missing fcv":             func(s *models.AcceptedState) { s.FCV = "" },
		"missing auth mechanisms": func(s *models.AcceptedState) { s.Auth.Mechanisms = nil },
		"no replica sets":         func(s *models.AcceptedState) { s.ReplicaSets = nil },
		"no members":              func(s *models.AcceptedState) { s.Members = nil },
		"mongod missing rs_member_id": func(s *models.AcceptedState) {
			m := s.Members["aaaaaaaa-0000-0000-0000-000000000001"]
			m.RSMemberID = nil
			s.Members["aaaaaaaa-0000-0000-0000-000000000001"] = m
		},
		"unknown process": func(s *models.AcceptedState) {
			m := s.Members["aaaaaaaa-0000-0000-0000-000000000001"]
			m.Process = "arbiter"
			s.Members["aaaaaaaa-0000-0000-0000-000000000001"] = m
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := validAcceptedState()
			mutate(s)
			_, err := models.MarshalAcceptedState(s)
			require.Error(t, err)
		})
	}

	t.Run("rejects a mongos carrying a replica-set field", func(t *testing.T) {
		s := validAcceptedState()
		m := validMongos()
		m.RSMemberID = new(int32(1))
		s.Members["bbbbbbbb-0000-0000-0000-000000000001"] = m
		_, err := models.MarshalAcceptedState(s)
		require.Error(t, err)
	})
}
