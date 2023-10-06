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
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/reform.v1"
)

func checkUniqueRuleID(q *reform.Querier, id string) error {
	if id == "" {
		panic("empty Rule ID")
	}

	rule := &Rule{ID: id}
	err := q.Reload(rule)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil
		}
		return errors.WithStack(err)
	}

	return status.Errorf(codes.AlreadyExists, "Rule with ID %q already exists.", id)
}

// FindRules returns saved alert rules configuration.
func FindRules(q *reform.Querier) ([]*Rule, error) {
	rows, err := q.SelectAllFrom(RuleTable, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to select alert rules")
	}

	rules := make([]*Rule, len(rows))
	for i, s := range rows {
		rules[i] = s.(*Rule) //nolint:forcetypeassert
	}

	return rules, nil
}

// FindRulesOnPage returns a page with saved alert rules configuration.
func FindRulesOnPage(q *reform.Querier, pageIndex, pageSize int) ([]*Rule, error) {
	rows, err := q.SelectAllFrom(RuleTable, "ORDER BY id LIMIT $1 OFFSET $2", pageSize, pageIndex*pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select alert rules")
	}

	rules := make([]*Rule, len(rows))
	for i, s := range rows {
		rules[i] = s.(*Rule) //nolint:forcetypeassert
	}

	return rules, nil
}

// CountRules returns number of alert rules.
func CountRules(q *reform.Querier) (int, error) {
	count, err := q.Count(RuleTable, "")
	if err != nil {
		return 0, errors.Wrap(err, "failed to count alert rules")
	}

	return count, nil
}

// FindRuleByID finds Rule by ID.
func FindRuleByID(q *reform.Querier, id string) (*Rule, error) {
	if id == "" {
		return nil, status.Error(codes.InvalidArgument, "Empty Rule ID.")
	}

	rule := &Rule{ID: id}
	err := q.Reload(rule)
	if err != nil {
		if errors.Is(err, reform.ErrNoRows) {
			return nil, status.Errorf(codes.NotFound, "Rule with ID %q not found.", id)
		}
		return nil, errors.WithStack(err)
	}

	return rule, nil
}

// CreateRuleParams are params for creating new Rule.
type CreateRuleParams struct {
	Name              string
	TemplateName      string
	Summary           string
	Disabled          bool
	ExprTemplate      string
	ParamsDefinitions AlertExprParamsDefinitions
	ParamsValues      AlertExprParamsValues
	DefaultFor        time.Duration
	For               time.Duration
	DefaultSeverity   Severity
	Severity          Severity
	CustomLabels      map[string]string
	Labels            map[string]string
	Annotations       map[string]string
	Filters           Filters
	ChannelIDs        []string
}

// CreateRule persists alert Rule.
func CreateRule(q *reform.Querier, params *CreateRuleParams) (*Rule, error) {
	id := "/rule_id/" + uuid.New().String()
	var err error
	if err = checkUniqueRuleID(q, id); err != nil {
		return nil, err
	}

	row := &Rule{
		ID:                id,
		Name:              params.Name,
		TemplateName:      params.TemplateName,
		Summary:           params.Summary,
		Disabled:          params.Disabled,
		ExprTemplate:      params.ExprTemplate,
		ParamsDefinitions: params.ParamsDefinitions,
		ParamsValues:      params.ParamsValues,
		DefaultFor:        params.DefaultFor,
		For:               params.For,
		DefaultSeverity:   params.DefaultSeverity,
		Severity:          params.Severity,
		Filters:           params.Filters,
	}

	if len(params.ChannelIDs) != 0 {
		channelIDs := deduplicateStrings(params.ChannelIDs)
		channels, err := FindChannelsByIDs(q, channelIDs)
		if err != nil {
			return nil, err
		}

		if len(channelIDs) != len(channels) {
			missingChannelsIDs := findMissingChannels(channelIDs, channels)
			return nil, status.Errorf(codes.NotFound, "Failed to find all required channels: %v.", missingChannelsIDs)
		}

		row.ChannelIDs = channelIDs
	}

	if err = row.SetCustomLabels(params.CustomLabels); err != nil {
		return nil, err
	}

	if err = row.SetLabels(params.Labels); err != nil {
		return nil, err
	}

	if err = row.SetAnnotations(params.Annotations); err != nil {
		return nil, err
	}

	if err = q.Insert(row); err != nil {
		return nil, errors.Wrap(err, "failed to create alert rule")
	}

	return row, nil
}

// ChangeRuleParams is params for updating existing Rule.
type ChangeRuleParams struct {
	Name         string
	Disabled     bool
	ParamsValues AlertExprParamsValues
	For          time.Duration
	Severity     Severity
	CustomLabels map[string]string
	Filters      Filters
	ChannelIDs   []string
}

// ChangeRule updates existing alerts Rule.
func ChangeRule(q *reform.Querier, ruleID string, params *ChangeRuleParams) (*Rule, error) {
	row, err := FindRuleByID(q, ruleID)
	if err != nil {
		return nil, err
	}

	channelIDs := deduplicateStrings(params.ChannelIDs)
	channels, err := FindChannelsByIDs(q, channelIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find channels")
	}

	if len(channelIDs) != len(channels) {
		missingChannelsIDs := findMissingChannels(channelIDs, channels)
		return nil, status.Errorf(codes.NotFound, "Failed to find all required channels: %v.", missingChannelsIDs)
	}

	row.Name = params.Name
	row.Disabled = params.Disabled
	row.For = params.For
	row.Severity = params.Severity
	row.Filters = params.Filters
	row.ParamsValues = params.ParamsValues

	labels, err := json.Marshal(params.CustomLabels)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update alert rule")
	}
	row.CustomLabels = labels
	row.ChannelIDs = params.ChannelIDs

	if err = q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to change alerts Rule")
	}

	return row, nil
}

// ToggleRuleParams represents rule toggle parameters.
type ToggleRuleParams struct {
	Disabled *bool // nil - do not change
}

// ToggleRule updates some alert rule fields.
func ToggleRule(q *reform.Querier, ruleID string, params *ToggleRuleParams) (*Rule, error) {
	row, err := FindRuleByID(q, ruleID)
	if err != nil {
		return nil, err
	}

	if params.Disabled == nil {
		return row, nil
	}

	row.Disabled = *params.Disabled

	if err = q.Update(row); err != nil {
		return nil, errors.Wrap(err, "failed to toggle alerts Rule")
	}

	return row, nil
}

// RemoveRule removes alert Rule with specified id.
func RemoveRule(q *reform.Querier, id string) error {
	var err error
	if _, err = FindRuleByID(q, id); err != nil {
		return err
	}

	if err = q.Delete(&Rule{ID: id}); err != nil {
		return errors.Wrap(err, "failed to delete alert Rule")
	}
	return nil
}

func findMissingChannels(ids []string, channels []*Channel) []string {
	m := make(map[string]bool, len(ids))
	for _, id := range ids {
		m[id] = false
	}

	for _, channel := range channels {
		if _, ok := m[channel.ID]; ok {
			m[channel.ID] = true
		}
	}

	var res []string
	for k, v := range m {
		if !v {
			res = append(res, k)
		}
	}

	return res
}
