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
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

const (
	stsGlobalRegion = "us-east-1"
	stsRequestBody  = "Action=GetCallerIdentity&Version=2011-06-15"
)

// awsLoginData returns the body of a Vault aws auth login with the iam type: a signed
// sts:GetCallerIdentity request that Vault forwards to AWS to learn the caller's identity.
// See https://developer.hashicorp.com/vault/api-docs/auth/aws#login.
func awsLoginData(
	ctx context.Context,
	provider aws.CredentialsProvider,
	region, serverID string,
	now time.Time,
) (map[string]any, error) {
	if provider == nil {
		return nil, errors.New("no AWS credentials found for vault login")
	}

	creds, err := provider.Retrieve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve AWS credentials for vault login: %w", err)
	}

	endpoint := "https://sts.amazonaws.com/"
	signingRegion := stsGlobalRegion
	if region != "" {
		endpoint = "https://sts." + region + ".amazonaws.com/"
		signingRegion = region
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(stsRequestBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	if serverID != "" {
		req.Header.Set("X-Vault-Aws-Iam-Server-Id", serverID)
	}

	hash := sha256.Sum256([]byte(stsRequestBody))
	err = v4.NewSigner().SignHTTP(ctx, creds, req, hex.EncodeToString(hash[:]), "sts", signingRegion, now)
	if err != nil {
		return nil, fmt.Errorf("failed to sign the AWS request for vault login: %w", err)
	}

	headers, err := json.Marshal(req.Header)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"iam_http_request_method": http.MethodPost,
		"iam_request_url":         base64.StdEncoding.EncodeToString([]byte(endpoint)),
		"iam_request_body":        base64.StdEncoding.EncodeToString([]byte(stsRequestBody)),
		"iam_request_headers":     base64.StdEncoding.EncodeToString(headers),
	}, nil
}
