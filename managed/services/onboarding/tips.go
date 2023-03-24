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

// Package onboarding provides functionality for user onboarding features.
package onboarding

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"gopkg.in/reform.v1"

	"github.com/percona/pmm/api/onboardingpb"
	"github.com/percona/pmm/managed/models"
)

const (
	InstallPMMServerTipID    = 1
	InstallPMMClientTipID    = 2
	ConnectServiceToPMMTipID = 3
)

type TipsService struct {
	db               *reform.DB
	inventoryService inventoryService

	systemTipIDs map[int32]struct{}

	onboardingpb.UnimplementedTipServiceServer
}

var _ onboardingpb.TipServiceServer = (*TipsService)(nil)

func NewTipService(db *reform.DB, inventoryService inventoryService) *TipsService {
	return &TipsService{
		db:               db,
		inventoryService: inventoryService,
		systemTipIDs: map[int32]struct{}{
			InstallPMMServerTipID:    {},
			InstallPMMClientTipID:    {},
			ConnectServiceToPMMTipID: {},
		},
	}
}

func (t *TipsService) GetTipStatus(ctx context.Context, tipRequest *onboardingpb.GetTipRequest) (*onboardingpb.GetTipResponse, error) {
	switch tipRequest.TipType {
	case onboardingpb.TipType_SYSTEM:
		return t.retrieveSystemTip(tipRequest.TipId)
	case onboardingpb.TipType_USER:
		return t.retrieveOrCreateUserTip(tipRequest)
	default:
		return nil, errors.New("Tip type is not correct")
	}
}

func (t *TipsService) retrieveOrCreateUserTip(tipRequest *onboardingpb.GetTipRequest) (*onboardingpb.GetTipResponse, error) {
	var tip models.UserTip
	err := t.db.InTransaction(func(tx *reform.TX) error {
		var err error
		tip, err = t.retrieveUserTip(tx, tipRequest.TipId, tipRequest.UserId)
		if err != nil {
			if err == reform.ErrNoRows {
				tip, err = t.createUserTip(tx, tipRequest.TipId, tipRequest.UserId)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("cannot create user tip by id: %d", tipRequest.TipId))
				}
			} else {
				return errors.Wrap(err, fmt.Sprintf("cannot retrieve user tip by id: %d", tipRequest.TipId))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &onboardingpb.GetTipResponse{
		TipId:       tip.UserTipID,
		IsCompleted: tip.IsCompleted,
	}, nil
}

func (t *TipsService) retrieveSystemTip(tipID int32) (*onboardingpb.GetTipResponse, error) {
	if ok := t.isSystemTip(tipID); !ok {
		return nil, errors.New(fmt.Sprintf("system tip doesn't exist: %d", tipID))
	}

	res, err := t.db.Querier.FindOneFrom(models.SystemTipTable, "id", tipID)
	if err != nil && err != reform.ErrNoRows {
		return nil, errors.Wrap(err, "failed to retrieve system tip by id")
	}
	var tip *models.SystemTip
	if err == reform.ErrNoRows {
		tip = &models.SystemTip{}
		tip.ID = tipID
	} else {
		tip = res.(*models.SystemTip)
	}

	if !tip.IsCompleted {
		isCompleted, err := t.isSystemTipCompleted(tip.ID)
		if err != nil {
			return nil, err
		}
		tip.IsCompleted = isCompleted

		err = t.db.Save(tip)
		if err != nil {
			return nil, errors.Wrap(err, "cannot save tip info")
		}
	}
	return &onboardingpb.GetTipResponse{
		TipId:       tip.ID,
		IsCompleted: tip.IsCompleted,
	}, nil
}

func (t *TipsService) isSystemTipCompleted(tipID int32) (bool, error) {
	switch tipID {
	case InstallPMMServerTipID:
		return true, nil
	case InstallPMMClientTipID:
		isCompleted, err := t.isAnyExternalClientConnected()
		if err != nil {
			return false, errors.Wrap(err, "Cannot retrieve list of agents to check the status of tip")
		}
		return isCompleted, nil
	case ConnectServiceToPMMTipID:
		isCompleted, err := t.isAnyServiceConnected()
		if err != nil {
			return false, errors.Wrap(err, "Cannot retrieve list of services to check the status of tip")
		}
		return isCompleted, nil
	default:
		return false, errors.Errorf("system tip doesn't exist: %d", tipID)
	}
}

func (t *TipsService) isAnyExternalClientConnected() (bool, error) {
	pmmServerAgentsByAgentID, err := models.FindAgents(t.db.Querier, models.AgentFilters{
		PMMAgentID: "pmm-server",
	})
	if err != nil {
		return false, errors.Wrap(err, "cannot find agents by agent-id 'pmm-server'")
	}

	pmmServerAgentsByNodeID, err := models.FindAgents(t.db.Querier, models.AgentFilters{
		NodeID: "pmm-server",
	})
	if err != nil {
		return false, errors.Wrap(err, "cannot find agents by node-id 'pmm-server'")
	}

	allPmmAgents, err := models.FindAgents(t.db.Querier, models.AgentFilters{})
	if err != nil {
		return false, errors.Wrap(err, "cannot find all agents")
	}

	return len(allPmmAgents) > (len(pmmServerAgentsByAgentID) + len(pmmServerAgentsByNodeID)), nil
}

func (t *TipsService) isAnyServiceConnected() (bool, error) {
	list, err := t.inventoryService.List(context.Background(), models.ServiceFilters{})
	if err != nil {
		return false, err
	}
	// after installation, we already have connected one service, it's PMM PostgresSQL
	// if we have second connected service then user already installed a second one
	return len(list) >= 2, nil
}

func (t *TipsService) retrieveUserTip(tx *reform.TX, tipID int32, userID int32) (models.UserTip, error) {
	res, err := tx.Querier.SelectOneFrom(models.UserTipTable, "WHERE user_id = $1 AND user_tip_id = $2", userID, tipID)
	if err != nil {
		if err == reform.ErrNoRows {
			return models.UserTip{}, err
		}
		return models.UserTip{}, errors.Wrap(err, "failed to retrieve system tip by id")
	}

	return *res.(*models.UserTip), nil
}

func (t *TipsService) createUserTip(tx *reform.TX, tipID int32, userID int32) (models.UserTip, error) {
	tip := models.UserTip{
		UserID:      userID,
		UserTipID:   tipID,
		IsCompleted: false,
	}
	err := tx.Save(&tip)
	if err != nil {
		return models.UserTip{}, err
	}
	return tip, nil
}

func (t *TipsService) CompleteUserTip(ctx context.Context, userTipRequest *onboardingpb.CompleteUserTipRequest) (*onboardingpb.CompleteUserTipResponse, error) {
	if ok := t.isSystemTip(userTipRequest.TipId); ok {
		return nil, errors.New("Tip ID is not correct, it's system tip")
	}
	err := t.db.InTransaction(func(tx *reform.TX) error {
		tip, err := t.retrieveUserTip(tx, userTipRequest.TipId, userTipRequest.UserId)
		if err != nil {
			if err == reform.ErrNoRows {
				tip, err = t.createUserTip(tx, userTipRequest.TipId, userTipRequest.UserId)
				if err != nil {
					return errors.Wrap(err, fmt.Sprintf("cannot create user tip by id: %d", userTipRequest.TipId))
				}
			} else {
				return errors.Wrap(err, fmt.Sprintf("cannot retrieve user tip by id: %d", userTipRequest.TipId))
			}
		}

		tip.IsCompleted = true

		err = tx.Save(&tip)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("cannot save user tip by id: %v", tip))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &onboardingpb.CompleteUserTipResponse{}, nil
}

func (t *TipsService) isSystemTip(tipID int32) bool {
	_, ok := t.systemTipIDs[tipID]
	return ok
}
