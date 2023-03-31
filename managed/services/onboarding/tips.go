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

func (t *TipsService) GetOnboardingStatus(ctx context.Context, tipRequest *onboardingpb.GetOnboardingStatusRequest) (*onboardingpb.GetOnboardingStatusResponse, error) {
	systemTips, err := t.retrieveSystemTips()
	if err != nil {
		return nil, err
	}

	userTips, err := t.retrieveUserTips(tipRequest.UserId)
	if err != nil {
		return nil, err
	}

	return &onboardingpb.GetOnboardingStatusResponse{
		SystemTips: systemTips,
		UserTips:   userTips,
	}, nil
}

func (t *TipsService) retrieveSystemTips() ([]*onboardingpb.TipModel, error) {
	structs, err := t.db.Querier.SelectAllFrom(models.OnboardingSystemTipTable, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrieve system tip by id")
	}
	tips := make([]*models.OnboardingSystemTip, len(structs))
	for i, s := range structs {
		tips[i] = s.(*models.OnboardingSystemTip)
	}

	for _, tip := range tips {
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
	}

	res := make([]*onboardingpb.TipModel, len(tips))
	for _, tip := range tips {
		res = append(res, &onboardingpb.TipModel{
			TipId:       tip.ID,
			IsCompleted: tip.IsCompleted,
		})
	}
	return res, nil
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

func (t *TipsService) retrieveUserTips(userID int32) ([]*onboardingpb.TipModel, error) {
	var res []*onboardingpb.TipModel
	err := t.db.InTransaction(func(tx *reform.TX) error {
		structs, err := tx.Querier.FindAllFrom(models.OnboardingTipTable, "type", "user")
		if err != nil {
			return err
		}

		var userTips []models.OnboardingUserTip
		for _, s := range structs {
			userTips = append(userTips, models.OnboardingUserTip{
				TipID:  (s.(*models.OnboardingTip)).ID,
				UserID: userID,
			})
		}

		for _, userTip := range userTips {
			retrievedUser, err := t.retrieveUserTip(tx, userTip.TipID, userID)
			if err != nil {
				if err == reform.ErrNoRows {
					retrievedUser, err = t.createUserTip(tx, userTip.TipID, userID)
					if err != nil {
						return err
					}
				} else {
					return errors.Wrap(err, "failed to retrieve system tip by id")
				}
			}
			res = append(res, &onboardingpb.TipModel{
				TipId:       retrievedUser.TipID,
				IsCompleted: retrievedUser.IsCompleted,
			})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (t *TipsService) retrieveUserTip(tx *reform.TX, tipID int32, userID int32) (*models.OnboardingUserTip, error) {
	res, err := tx.Querier.SelectOneFrom(models.OnboardingUserTipTable, "WHERE user_id = $1 AND tip_id = $2", userID, tipID)
	if err != nil {
		if err == reform.ErrNoRows {
			return nil, err
		}
		return nil, errors.Wrap(err, "failed to retrieve system tip by id")
	}

	return res.(*models.OnboardingUserTip), nil
}

func (t *TipsService) createUserTip(tx *reform.TX, tipID int32, userID int32) (*models.OnboardingUserTip, error) {
	tip := &models.OnboardingUserTip{
		UserID:      userID,
		TipID:       tipID,
		IsCompleted: false,
	}
	err := tx.Save(tip)
	if err != nil {
		return nil, err
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
				return errors.Wrap(err, "cannot complete because tip is not found")
			}
			return errors.Wrap(err, "cannot complete user tip")
		}

		if tip.IsCompleted {
			return errors.New("cannot complete tip because it's already completed!")
		}

		tip.IsCompleted = true
		err = tx.Save(tip)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("cannot save user tip by id: %v", *tip))
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
