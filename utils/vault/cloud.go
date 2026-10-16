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

package vault

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// The Azure Instance Metadata Service.
	azureMetadataURL = "http://169.254.169.254/metadata"
	// The metadata server of Google Compute Engine.
	gcpMetadataURL = "http://metadata.google.internal/computeMetadata/v1"

	// DefaultAzureResource is the resource that the access token of the managed identity is requested for,
	// which must be the one that the azure auth method of Vault is configured with.
	DefaultAzureResource = "https://management.azure.com/"

	metadataTimeout = 10 * time.Second
)

// metadataHTTPClient asks the metadata services of the clouds, which must be reached directly, never through a proxy.
var metadataHTTPClient = &http.Client{ //nolint:gochecknoglobals
	Transport: &http.Transport{Proxy: nil},
	Timeout:   metadataTimeout,
}

// metadataGet returns the body of a GET request to a metadata service, sending header with it.
func metadataGet(ctx context.Context, u string, header http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header = header

	resp, err := metadataHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	b, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %d %s", req.URL.Redacted(), resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	return b, nil
}

// azureLoginData returns the body of a Vault azure auth login: an access token of the managed identity of the
// virtual machine, and the virtual machine or scale set that it runs as, from the Azure Instance Metadata Service.
// See https://developer.hashicorp.com/vault/api-docs/auth/azure#login.
func azureLoginData(ctx context.Context, metadataURL, resource, clientID string) (map[string]any, error) {
	header := http.Header{"Metadata": []string{"true"}}

	q := url.Values{"api-version": []string{"2018-02-01"}, "resource": []string{resource}}
	if clientID != "" {
		// The user-assigned managed identity to use, rather than the system-assigned one.
		q.Set("client_id", clientID)
	}
	b, err := metadataGet(ctx, metadataURL+"/identity/oauth2/token?"+q.Encode(), header)
	if err != nil {
		return nil, fmt.Errorf("failed to get an access token of the managed identity for vault login: %w", err)
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	err = json.Unmarshal(b, &token)
	if err == nil && token.AccessToken == "" {
		err = errors.New("no access token in the response")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get an access token of the managed identity for vault login: %w", err)
	}

	b, err = metadataGet(ctx, metadataURL+"/instance/compute?api-version=2021-02-01", header)
	if err != nil {
		return nil, fmt.Errorf("failed to get the virtual machine from the Azure metadata for vault login: %w", err)
	}
	var compute struct {
		SubscriptionID    string `json:"subscriptionId"`
		ResourceGroupName string `json:"resourceGroupName"`
		Name              string `json:"name"`
		VMScaleSetName    string `json:"vmScaleSetName"`
	}
	err = json.Unmarshal(b, &compute)
	if err != nil {
		return nil, fmt.Errorf("failed to get the virtual machine from the Azure metadata for vault login: %w", err)
	}

	data := map[string]any{
		"jwt":                 token.AccessToken,
		"subscription_id":     compute.SubscriptionID,
		"resource_group_name": compute.ResourceGroupName,
	}
	if compute.VMScaleSetName != "" {
		data["vmss_name"] = compute.VMScaleSetName
	} else {
		data["vm_name"] = compute.Name
	}
	return data, nil
}

// gcpLoginData returns the body of a Vault gcp auth login with the gce type: an identity token of the service
// account of the instance, for the audience of the role, from the metadata server of Google Compute Engine.
// See https://developer.hashicorp.com/vault/docs/auth/gcp#gce-login.
func gcpLoginData(ctx context.Context, metadataURL, role string) (map[string]any, error) {
	q := url.Values{"audience": []string{"http://vault/" + role}, "format": []string{"full"}}
	b, err := metadataGet(ctx, metadataURL+"/instance/service-accounts/default/identity?"+q.Encode(),
		http.Header{"Metadata-Flavor": []string{"Google"}})
	if err == nil && len(b) == 0 {
		err = errors.New("no identity token in the response")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get an identity token of the instance for vault login: %w", err)
	}

	return map[string]any{"jwt": string(b)}, nil
}
