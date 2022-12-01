// dbaas-controller
// Copyright (C) 2020 Percona LLC
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
package olm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestGetLatestVersion(t *testing.T) {
// 	kubeconfig, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
// 	require.NoError(t, err)
//
// 	ctx := context.Background()
//
// 	client, err := k8sclient.New(ctx, string(kubeconfig))
// 	assert.NoError(t, err)
// 	defer client.Cleanup() //nolint:errcheck
//
// 	latest, err := getLatestVersion(ctx, olmRepo)
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, latest)
// }

func TestInstallOlmOperator(t *testing.T) {
	// kubeconfig, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	// require.NoError(t, err)

	ctx := context.Background()

	// client, err := k8sclient.New(ctx, string(kubeconfig))
	// assert.NoError(t, err)

	// t.Cleanup(func() {
	// 	// Maintain the order, otherwise the Kubernetes deletetion will stuck in Terminating state.
	// 	err := client.Delete(ctx, []string{"apiservices.apiregistration.k8s.io", "v1.packages.operators.coreos.com"})
	// 	assert.NoError(t, err)
	// 	files := []string{
	// 		"deploy/olm/crds.yaml",
	// 		"deploy/olm/olm.yaml",
	// 	}

	// 	for _, file := range files {
	// 		t.Logf("deleting %q\n", file)
	// 		yamlFile, _ := dbaascontroller.DeployDir.ReadFile(file)
	// 		// When deleting, some resources might be already deleted by the previous file so the returned error
	// 		// should be considered only as a warning.
	// 		_ = client.Delete(ctx, yamlFile)
	// 	}

	// 	_ = client.Cleanup()
	// })

	// req := &controllerv1beta1.InstallOLMOperatorRequest{
	// 	KubeAuth: &controllerv1beta1.KubeAuth{
	// 		Kubeconfig: string(kubeconfig),
	// 	},
	// }

	olms := NewOperatorService()
	err := olms.InstallOLMOperator(ctx)
	assert.NoError(t, err)

	// 	// Wait for the deployments
	// 	_, err = client.Run(ctx, []string{"rollout", "status", "-w", "deployment/olm-operator", "-n", "olm"})
	// 	assert.NoError(t, err)
	// 	_, err = client.Run(ctx, []string{"rollout", "status", "-w", "deployment/catalog-operator", "-n", "olm"})
	// 	assert.NoError(t, err)
	//
	// 	catalosSourceNamespace := "olm"
	// 	operatorName := "percona-server-mongodb-operator"
	//
	// 	manifests, err := olms.AvailableOperators(ctx, client, catalosSourceNamespace, operatorName)
	// 	assert.NoError(t, err)
	// 	assert.Equal(t, 1, len(manifests.Items))
	//
	// 	subscriptionName := "percona-server-mongodb-operator"
	// 	subscriptionNamespace := fmt.Sprintf("test-namespace-%04d", rand.Int63n(9999))
	// 	subscriptionNamespace = "default"
	// 	operatorGroup := "percona-operators-group"
	//
	// 	t.Run("Subscribe", func(t *testing.T) {
	// 		kubeconfig, err := ioutil.ReadFile(os.Getenv("HOME") + "/.kube/config")
	// 		require.NoError(t, err)
	//
	// 		ctx := context.Background()
	//
	// 		client, err := k8sclient.New(ctx, string(kubeconfig))
	// 		assert.NoError(t, err)
	// 		defer client.Cleanup() //nolint:errcheck
	//
	// 		// Install PSMDB Operator
	// 		params := &controllerv1beta1.InstallOperatorRequest{
	// 			KubeAuth: &controllerv1beta1.KubeAuth{
	// 				Kubeconfig: string(kubeconfig),
	// 			},
	// 			Namespace:              subscriptionNamespace,
	// 			Name:                   subscriptionName,
	// 			OperatorGroup:          operatorGroup,
	// 			CatalogSource:          "operatorhubio-catalog",
	// 			CatalogSourceNamespace: catalosSourceNamespace,
	// 			Channel:                "stable",
	// 			InstallPlanApproval:    string(v1alpha1.ApprovalManual),
	// 			StartingCsv:            "percona-server-mongodb-operator.v1.11.0",
	// 		}
	//
	// 		_, err = olms.InstallOperator(ctx, params)
	// 		assert.NoError(t, err)
	//
	// 		var subscription *controllerv1beta1.GetSubscriptionResponse
	//
	// 		subscription, err = olms.GetSubscription(ctx, &controllerv1beta1.GetSubscriptionRequest{
	// 			KubeAuth: &controllerv1beta1.KubeAuth{
	// 				Kubeconfig: string(kubeconfig),
	// 			},
	// 			Name:      subscriptionName,
	// 			Namespace: subscriptionNamespace,
	// 		})
	//
	// 		approveRequest := &controllerv1beta1.ApproveInstallPlanRequest{
	// 			KubeAuth: &controllerv1beta1.KubeAuth{
	// 				Kubeconfig: string(kubeconfig),
	// 			},
	// 			Namespace: subscriptionNamespace,
	// 			Name:      subscription.Subscription.InstallPlanName,
	// 		}
	//
	// 		_, err = olms.ApproveInstallPlan(ctx, approveRequest)
	// 		assert.NoError(t, err)
	//
	// 		t.Log("Waiting for deployment")
	// 		// Loop until the deployment exists and THEN we can wait.
	// 		// Sometimes the test reaches this point too fast and we get an error saying that the
	// 		// deplyment doesn't exists.
	// 		for i := 0; i < 5; i++ {
	// 			_, err = client.Run(ctx, []string{"wait", "--for=condition=Available", "deployment", params.Name, "-n", params.Namespace, "--timeout=380s"})
	// 			if err == nil {
	// 				t.Logf("Deployment ready at try number %d", i)
	// 				break
	// 			}
	// 			time.Sleep(30 * time.Second)
	// 		}
	// 		assert.NoError(t, err)
	//
	// 		// We installed PSMDB operator 1.11 but in the catalog, 1.12 is already available.
	// 		// There must be a new and unapproved install plan to upgrade the operator to 1.12.
	// 		ipListReq := &controllerv1beta1.ListInstallPlansRequest{
	// 			KubeAuth: &controllerv1beta1.KubeAuth{
	// 				Kubeconfig: string(kubeconfig),
	// 			},
	// 			Namespace:       subscriptionNamespace,
	// 			Name:            subscriptionName,
	// 			NotApprovedOnly: true,
	// 		}
	// 		installPlansForUpgrade, err := olms.ListInstallPlans(ctx, ipListReq)
	// 		assert.NoError(t, err)
	// 		assert.True(t, len(installPlansForUpgrade.Items) > 0)
	//
	// 		err = client.Delete(ctx, []string{"subscription", subscriptionName, "-n", params.Namespace})
	// 		assert.NoError(t, err)
	// 		err = client.Delete(ctx, []string{"operatorgroup", params.OperatorGroup, "-n", params.Namespace})
	// 		assert.NoError(t, err)
	// 		err = client.Delete(ctx, []string{"deployment", params.Name, "-n", params.Namespace})
	// 		assert.NoError(t, err)
	//
	// 		if params.Namespace != "default" {
	// 			err = client.Delete(ctx, []string{"namespace", params.Namespace})
	// 			assert.NoError(t, err)
	// 		}
	//	})
}
