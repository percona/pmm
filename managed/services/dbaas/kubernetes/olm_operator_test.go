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

// Package operator contains logic related to kubernetes operators.
package kubernetes

import (
	"context"
	"testing"

	v1 "github.com/operator-framework/api/pkg/operators/v1"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/percona/pmm/managed/services/dbaas/kubernetes/client"
)

func TestInstallOlmOperator(t *testing.T) {
	ctx := context.Background()
	k8sclient := &client.MockKubeClientConnector{}

	olms := NewEmpty()
	olms.client = k8sclient

	t.Run("Install OLM Operator", func(t *testing.T) {
		k8sclient.On("CreateSubscriptionForCatalog", mock.Anything, mock.Anything, mock.Anything,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
			Return(&v1alpha1.Subscription{}, nil)
		k8sclient.On("GetDeployment", ctx, mock.Anything).Return(&appsv1.Deployment{}, nil)
		k8sclient.On("ApplyFile", mock.Anything).Return(nil)
		k8sclient.On("DoRolloutWait", ctx, mock.Anything).Return(nil)
		k8sclient.On("GetSubscriptionCSV", ctx, mock.Anything).Return(types.NamespacedName{}, nil)
		k8sclient.On("DoRolloutWait", ctx, mock.Anything).Return(nil)
		err := olms.InstallOLMOperator(ctx)
		assert.NoError(t, err)
	})

	t.Run("Install PSMDB Operator", func(t *testing.T) {
		// Install PSMDB Operator
		subscriptionNamespace := "default"
		operatorGroup := "percona-operators-group"
		catalosSourceNamespace := "olm"
		operatorName := "percona-server-mongodb-operator"
		params := InstallOperatorRequest{
			Namespace:              subscriptionNamespace,
			Name:                   operatorName,
			OperatorGroup:          operatorGroup,
			CatalogSource:          "operatorhubio-catalog",
			CatalogSourceNamespace: catalosSourceNamespace,
			Channel:                "stable",
			InstallPlanApproval:    v1alpha1.ApprovalManual,
		}

		k8sclient.On("GetOperatorGroup", ctx, "", operatorGroup).Return(&v1.OperatorGroup{}, nil)
		mockSubscription := &v1alpha1.Subscription{
			Status: v1alpha1.SubscriptionStatus{
				Install: &v1alpha1.InstallPlanReference{
					Name: "abcd1234",
				},
			},
		}
		k8sclient.On("GetSubscription", ctx, subscriptionNamespace, operatorName).Return(mockSubscription, nil)
		mockInstallPlan := &v1alpha1.InstallPlan{}
		k8sclient.On("GetInstallPlan", ctx, subscriptionNamespace, mockSubscription.Status.Install.Name).Return(mockInstallPlan, nil)
		k8sclient.On("UpdateInstallPlan", ctx, subscriptionNamespace, mockInstallPlan).Return(mockInstallPlan, nil)
		err := olms.InstallOperator(ctx, params)
		assert.NoError(t, err)
	})
}
