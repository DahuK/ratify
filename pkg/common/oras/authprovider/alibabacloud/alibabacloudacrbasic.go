/*
Copyright The Ratify Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package alibabacloud

import (
	"context"
	"encoding/json"
	"fmt"
	cr20181201 "github.com/alibabacloud-go/cr-20181201/v2/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/credentials-go/credentials"
	re "github.com/ratify-project/ratify/errors"
	"os"
	"time"

	"github.com/pkg/errors"
	provider "github.com/ratify-project/ratify/pkg/common/oras/authprovider"
	"github.com/sirupsen/logrus"
)

const (
	EnvRoleArn              = "ALIBABA_CLOUD_ROLE_ARN"
	EnvOidcProviderArn      = "ALIBABA_CLOUD_OIDC_PROVIDER_ARN"
	EnvOidcTokenFile        = "ALIBABA_CLOUD_OIDC_TOKEN_FILE"
	AlibabaCloudACREndpoint = "cr.%s.aliyuncs.com"
)

type AlibabaCloudAcrBasicProviderFactory struct{} //nolint:revive // ignore linter to have unique type name

type alibabacloudAcrBasicAuthProvider struct {
	acrAuthToken AcrAuthToken
	instanceID   string
	providerName string
}

type alibabacloudAcrBasicAuthProviderConf struct {
	Name       string `json:"name"`
	InstanceID string `json:"instanceID"`
}

const (
	alibabacloudAcrAuthProviderName string = "alibabacloudAcrBasic"
)

// init calls Register for AlibabaCloud RRSA Basic Auth provider
func init() {
	provider.Register(alibabacloudAcrAuthProviderName, &AlibabaCloudAcrBasicProviderFactory{})
}

// Get ACR auth token from RRSA config
func (d *alibabacloudAcrBasicAuthProvider) getAcrAuthToken(artifact string) (AcrAuthToken, error) {
	// Verify RRSA ENV is present

	envRoleArn := os.Getenv(EnvRoleArn)
	envOidcProviderArn := os.Getenv(EnvOidcProviderArn)
	envOidcTokenFile := os.Getenv(EnvOidcTokenFile)

	if envRoleArn == "" || envOidcProviderArn == "" || envOidcTokenFile == "" {
		return AcrAuthToken{}, fmt.Errorf("required environment variables not set, ALIBABA_CLOUD_ROLE_ARN: %s, ALIBABA_CLOUD_OIDC_PROVIDER_ARN: %s, ALIBABA_CLOUD_OIDC_TOKEN_FILE: %s", envRoleArn, envOidcProviderArn, envOidcTokenFile)
	}

	cred, err := credentials.NewCredential(nil)
	config := &openapi.Config{
		Credential: cred,
	}

	// registry/region from image
	registry, err := provider.GetRegistryHostName(artifact)
	if err != nil {
		return AcrAuthToken{}, fmt.Errorf("failed to get registry from image: %w", err)
	}
	registryMetaInfo, err := getRegionFromArtifict(registry)
	if err != nil || registryMetaInfo.Region == "" {
		return AcrAuthToken{}, fmt.Errorf("failed to get region from image: %w", err)
	}
	region := registryMetaInfo.Region
	logrus.Debugf("Alibaba Cloud ACR basic artifact=%s, registry=%s, region=%s", artifact, registry, region)

	// Endpoint refer to https://help.aliyun.com/zh/acr/developer-reference/api-cr-2018-12-01-endpoint
	config.Endpoint = tea.String(fmt.Sprintf(AlibabaCloudACREndpoint, region))
	config.RegionId = tea.String(region)
	crClient, err := cr20181201.NewClient(config)
	if err != nil {
		return AcrAuthToken{}, fmt.Errorf("failed to init alibaba cloud acr client: %w", err)
	}

	getAuthorizationTokenRequest := &cr20181201.GetAuthorizationTokenRequest{}
	getAuthorizationTokenRequest.InstanceId = tea.String(d.instanceID)
	runtime := &util.RuntimeOptions{}

	tokenResponse, err := crClient.GetAuthorizationTokenWithOptions(getAuthorizationTokenRequest, runtime)
	if err != nil {
		return AcrAuthToken{}, fmt.Errorf("failed to init alibaba cloud acr client: %w", err)
	}
	d.acrAuthToken.AuthData[registry] = tokenResponse.Body

	return d.acrAuthToken, nil
}

// Create returns an Alibaba CloudAcrBasicProvider
func (s *AlibabaCloudAcrBasicProviderFactory) Create(authProviderConfig provider.AuthProviderConfig) (provider.AuthProvider, error) {
	conf := alibabacloudAcrBasicAuthProviderConf{}
	authProviderConfigBytes, err := json.Marshal(authProviderConfig)
	if err != nil {
		return nil, err
	}

	if err = json.Unmarshal(authProviderConfigBytes, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse auth provider configuration: %w", err)
	}
	//get ACR EE instance id
	instanceID := conf.InstanceID
	if instanceID == "" {
		instanceID = os.Getenv("ALIBABA_CLOUD_ACR_INSTANCE_ID")
		if instanceID == "" {
			return nil, re.ErrorCodeEnvNotSet.WithComponentType(re.AuthProvider).WithDetail("no instance ID provided and ALIBABA_CLOUD_ACR_INSTANCE_ID environment variable is empty")
		}
	}

	acrAuthToken := AcrAuthToken{}
	acrAuthToken.AuthData = make(map[string]*cr20181201.GetAuthorizationTokenResponseBody)

	return &alibabacloudAcrBasicAuthProvider{
		acrAuthToken: acrAuthToken,
		instanceID:   instanceID,
		providerName: alibabacloudAcrAuthProviderName,
	}, nil
}

// Enabled checks for non-empty AlibabaCloud RAM creds
func (d *alibabacloudAcrBasicAuthProvider) Enabled(_ context.Context) bool {
	if d.providerName == "" {
		logrus.Error("basic Alibaba Cloud ACR providerName was empty")
		return false
	}

	return true
}

// Provide returns the credentials for a specified artifact.
// Uses AlibabaCloud RRSA to retrieve creds from RRSA credential chain
func (d *alibabacloudAcrBasicAuthProvider) Provide(ctx context.Context, artifact string) (provider.AuthConfig, error) {
	logrus.Debugf("artifact = %s", artifact)

	if !d.Enabled(ctx) {
		return provider.AuthConfig{}, fmt.Errorf("Alibaba Cloud RRSA basic auth provider is not properly enabled")
	}

	registry, err := provider.GetRegistryHostName(artifact)
	if err != nil {
		return provider.AuthConfig{}, errors.Wrapf(err, "could not get ACR registry from %s", artifact)
	}

	if !d.acrAuthToken.exists(registry) {
		logrus.Debugf("acrAuthToken for %s does not exist", registry)
		_, err = d.getAcrAuthToken(artifact)
		if err != nil {
			return provider.AuthConfig{}, errors.Wrapf(err, "could not get ACR auth token for %s", artifact)
		}
	}

	// need to refresh AlibabaCloud ACR credentials
	t := time.Now().Add(time.Minute * 5)
	if t.After(d.acrAuthToken.Expiry(registry)) || time.Now().After(d.acrAuthToken.Expiry(registry)) {
		_, err = d.getAcrAuthToken(artifact)
		if err != nil {
			return provider.AuthConfig{}, errors.Wrapf(err, "could not refresh ACR auth token for %s", artifact)
		}

		logrus.Debugf("successfully refreshed ACR auth token for %s", artifact)
	}
	// Get ACR basic auth creds from AuthData response
	tmpUsr := tea.StringValue(d.acrAuthToken.AuthData[registry].TempUsername)
	passwd := tea.StringValue(d.acrAuthToken.AuthData[registry].AuthorizationToken)
	authConfig := provider.AuthConfig{
		Username:  tmpUsr,
		Password:  passwd,
		Provider:  d,
		ExpiresOn: d.acrAuthToken.Expiry(registry),
	}

	return authConfig, nil
}
