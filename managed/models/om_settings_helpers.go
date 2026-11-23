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
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"gopkg.in/reform.v1"
)

// OmEffectiveSetting is one key's resolved value for a scope. For a secret key Value is nil and
// only IsSet and Source are meaningful, so the payload never leaves storage.
type OmEffectiveSetting struct {
	Key         string
	Value       json.RawMessage
	Source      OmSettingSource
	HasOverride bool
	Secret      bool
	IsSet       bool
	Description string
}

// scopeOf returns the scope a clusterID selects.
func scopeOf(clusterID *string) OmSettingScope {
	if clusterID == nil {
		return OmSettingGlobal
	}
	return OmSettingCluster
}

// GetOmSettings resolves every registered key for a scope: a cluster override wins over the global
// value, which wins over the registry default. A secret key reports only set/unset and its source.
func GetOmSettings(q *reform.Querier, reg *OmSettingsRegistry, clusterID *string) ([]OmEffectiveSetting, error) {
	global, err := settingRows(q, nil)
	if err != nil {
		return nil, err
	}
	cluster := map[string]json.RawMessage{}
	if clusterID != nil {
		cluster, err = settingRows(q, clusterID)
		if err != nil {
			return nil, err
		}
	}

	out := make([]OmEffectiveSetting, 0, len(reg.defs))
	for _, key := range reg.Keys() {
		def, _ := reg.Def(key)
		eff := OmEffectiveSetting{Key: key, Secret: def.Secret, Description: def.Description}
		clusterVal, hasCluster := cluster[key]
		globalVal, hasGlobal := global[key]
		switch {
		case clusterID != nil && hasCluster:
			eff.Source = OmSettingSourceCluster
			eff.Value = clusterVal
		case hasGlobal:
			eff.Source = OmSettingSourceGlobal
			eff.Value = globalVal
		default:
			eff.Source = OmSettingSourceDefault
			eff.Value = def.Default
		}
		eff.HasOverride = eff.Source != OmSettingSourceDefault
		eff.IsSet = eff.HasOverride
		if def.Secret {
			eff.Value = nil
		}
		out = append(out, eff)
	}
	return out, nil
}

// settingRows reads the stored values for one scope keyed by setting key.
func settingRows(q *reform.Querier, clusterID *string) (map[string]json.RawMessage, error) {
	var rows *sql.Rows
	var err error
	if clusterID == nil {
		rows, err = q.Query("SELECT key, value FROM om_settings WHERE cluster_id IS NULL")
	} else {
		rows, err = q.Query("SELECT key, value FROM om_settings WHERE cluster_id = $1", *clusterID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read settings: %w", err)
	}
	defer rows.Close() //nolint:errcheck
	out := map[string]json.RawMessage{}
	for rows.Next() {
		var key string
		var value []byte
		err = rows.Scan(&key, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		out[key] = value
	}
	return out, rows.Err()
}

// UpdateOmSettings validates the whole batch, then stores each value and writes one
// settings_change operation in a single transaction; a secret value is stored encrypted by reference.
func UpdateOmSettings(
	ctx context.Context, db *reform.DB, reg *OmSettingsRegistry, clusterID *string,
	principal json.RawMessage, who string, values map[string]json.RawMessage,
) error {
	if len(values) == 0 {
		return NewInvalidArgumentError("at least one value is required")
	}
	scope := scopeOf(clusterID)
	keys := make([]string, 0, len(values))
	for key := range values {
		_, err := reg.validateValue(key, scope, values[key])
		if err != nil {
			return err
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		for _, key := range keys {
			def, _ := reg.Def(key)
			stored := values[key]
			if def.Secret {
				ref, err := storeSettingSecret(tx.Querier, clusterID, key, values[key])
				if err != nil {
					return err
				}
				stored = ref
			}
			err := upsertSetting(tx.Querier, clusterID, key, stored, who)
			if err != nil {
				return err
			}
		}
		return recordSettingsChange(tx.Querier, clusterID, principal, who, "update", keys)
	})
}

// DeleteOmSettingOverride removes one scope's override. A delete of an absent override is a no-op
// that records nothing. Returns whether an override was removed.
func DeleteOmSettingOverride(
	ctx context.Context, db *reform.DB, reg *OmSettingsRegistry, clusterID *string,
	who string, principal json.RawMessage, key string,
) (bool, error) {
	_, ok := reg.Def(key)
	if !ok {
		return false, fmt.Errorf("settings: unknown key '%s'", key)
	}
	removed := false
	err := db.InTransactionContext(ctx, nil, func(tx *reform.TX) error {
		res, err := deleteSettingRow(tx.Querier, clusterID, key)
		if err != nil {
			return err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if n == 0 {
			return nil
		}
		removed = true
		def, _ := reg.Def(key)
		if def.Secret {
			err = retireSettingSecret(tx.Querier, clusterID, key)
			if err != nil {
				return err
			}
		}
		return recordSettingsChange(tx.Querier, clusterID, principal, who, "delete", []string{key})
	})
	return removed, err
}

// storeSettingSecret retires the current setting secret for a scope and stores the new value
// encrypted, returning the {"secret_id": ...} reference for om_settings.value.
func storeSettingSecret(q *reform.Querier, clusterID *string, key string, value json.RawMessage) (json.RawMessage, error) {
	err := retireSettingSecret(q, clusterID, key)
	if err != nil {
		return nil, err
	}
	discriminator := key
	secret := &OmSecret{
		Purpose: "setting", Discriminator: &discriminator, ClusterID: clusterID,
		Generation: OmSecretCurrent, Payload: string(value),
	}
	err = CreateOmSecret(q, secret)
	if err != nil {
		return nil, err
	}
	ref, err := json.Marshal(map[string]string{"secret_id": secret.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret reference: %w", err)
	}
	return ref, nil
}

// retireSettingSecret marks the current setting secret for a scope retired, so the one-current
// index admits the replacement.
const retireSettingSecretSQL = "UPDATE om_secrets SET generation = 'retired', rotated_at = now()" +
	" WHERE purpose = 'setting' AND discriminator = $1 AND generation = 'current'"

func retireSettingSecret(q *reform.Querier, clusterID *string, key string) error {
	var err error
	if clusterID == nil {
		_, err = q.Exec(retireSettingSecretSQL+" AND cluster_id IS NULL AND operation_id IS NULL", key)
	} else {
		_, err = q.Exec(retireSettingSecretSQL+" AND cluster_id = $2 AND replica_set_ref IS NULL", key, *clusterID)
	}
	if err != nil {
		return fmt.Errorf("failed to retire setting secret: %w", err)
	}
	return nil
}

// upsertSetting writes a scope's stored value for one key.
func upsertSetting(q *reform.Querier, clusterID *string, key string, value json.RawMessage, who string) error {
	res, err := settingUpdate(q, clusterID, key, value, who)
	if err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	row := &OmSetting{Key: key, ClusterID: clusterID, Value: value, UpdatedBy: who}
	err = q.Insert(row)
	if err != nil {
		return fmt.Errorf("failed to insert setting: %w", err)
	}
	return nil
}

func settingUpdate(q *reform.Querier, clusterID *string, key string, value json.RawMessage, who string) (sql.Result, error) {
	if clusterID == nil {
		return q.Exec("UPDATE om_settings SET value = $1, updated_by = $2, updated_at = now() WHERE key = $3 AND cluster_id IS NULL", []byte(value), who, key)
	}
	return q.Exec("UPDATE om_settings SET value = $1, updated_by = $2, updated_at = now() WHERE key = $3 AND cluster_id = $4", []byte(value), who, key, *clusterID)
}

func deleteSettingRow(q *reform.Querier, clusterID *string, key string) (sql.Result, error) {
	if clusterID == nil {
		return q.Exec("DELETE FROM om_settings WHERE key = $1 AND cluster_id IS NULL", key)
	}
	return q.Exec("DELETE FROM om_settings WHERE key = $1 AND cluster_id = $2", key, *clusterID)
}

// recordSettingsChange writes one succeeded settings_change operation, with params holding the verb,
// keys and scope and never a secret value.
func recordSettingsChange(q *reform.Querier, clusterID *string, principal json.RawMessage, who, verb string, keys []string) error {
	params, err := json.Marshal(map[string]any{"verb": verb, "keys": keys, "scope": scopeOf(clusterID)})
	if err != nil {
		return fmt.Errorf("failed to marshal settings_change params: %w", err)
	}
	if principal == nil {
		p, merr := json.Marshal(map[string]string{"kind": "user", "login": who})
		if merr != nil {
			return fmt.Errorf("failed to marshal principal: %w", merr)
		}
		principal = p
	}
	now := Now()
	op := &OmOperation{
		ID: uuid.New().String(), Kind: "settings_change", KindVersion: 1, ClusterID: clusterID,
		Status: OmOperationSucceeded, Trigger: OmTriggerUser, Principal: principal, Params: params,
		FinishedAt: &now,
	}
	err = q.Insert(op)
	if err != nil {
		return fmt.Errorf("failed to record settings_change: %w", err)
	}
	return nil
}
