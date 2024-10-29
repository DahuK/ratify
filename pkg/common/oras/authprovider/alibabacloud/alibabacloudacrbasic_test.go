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
	"encoding/base64"
	cr20181201 "github.com/alibabacloud-go/cr-20181201/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	// #nosec G101 (Ref: https://github.com/securego/gosec/issues/295)
	testPassword              = "eyJwYXlsb2FkIjoiOThPNTFqemhaUmZWVG"
	testUserMeta              = `{"instanceId":"cri-xxxxxxx","time":"1730185474000","type":"user","userId":"123456"}`
	testHost                  = "test-registry.cn-hangzhou.cr.aliyuncs.com"
	testArtifact              = testHost + "/foo:latest"
	testArtifactWithoutRegion = "test-registry-vpc.cr.aliyuncs.com/foo/test:latest"
)

// Verifies that alibabacloudAcrBasicAuthProvider is properly constructed by mocking AcrAuthToken

func mockAuthProvider() alibabacloudAcrBasicAuthProvider {
	// Setup
	acrAuthToken := AcrAuthToken{}
	acrAuthToken.AuthData = make(map[string]*cr20181201.GetAuthorizationTokenResponseBody)

	creds := []string{testUserMeta, testPassword}
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(creds, ":")))

	expiry := time.Now().Add(time.Minute * 10)
	acrAuthToken.AuthData[testHost] = &cr20181201.GetAuthorizationTokenResponseBody{
		AuthorizationToken: tea.String(encoded),
		ExpireTime:         tea.Int64(expiry.UnixMilli()),
	}

	return alibabacloudAcrBasicAuthProvider{
		acrAuthToken: acrAuthToken,
		providerName: alibabacloudAcrAuthProviderName,
	}
}

func TestAlibabaCloudAcrBasicAuthProvider_Create(t *testing.T) {
	authProviderConfig := map[string]interface{}{
		"name":       "alibabacloudAcrBasic",
		"instanceID": "cri-testing",
	}

	factory := AlibabaCloudAcrBasicProviderFactory{}
	_, err := factory.Create(authProviderConfig)

	if err != nil {
		t.Fatalf("create failed %v", err)
	}
}

func TestAlibabaCloudAcrBasicAuthProvider_Enabled(t *testing.T) {
	authProvider := mockAuthProvider()

	ctx := context.Background()

	if !authProvider.Enabled(ctx) {
		t.Fatal("enabled should have returned true but returned false")
	}

	authProvider.providerName = ""
	if authProvider.Enabled(ctx) {
		t.Fatal("enabled should have returned false but returned true")
	}
}

func TestAlibabaCloudAcrBasicAuthProvider_ProvidesWithArtifact(t *testing.T) {
	authProvider := mockAuthProvider()

	_, err := authProvider.Provide(context.TODO(), testArtifact)
	if err != nil {
		t.Fatalf("encountered error: %+v", err)
	}
}

func TestAlibabaCloudAcrBasicAuthProvider_ProvidesWithHost(t *testing.T) {
	authProvider := mockAuthProvider()

	_, err := authProvider.Provide(context.TODO(), testHost)
	if err != nil {
		t.Fatalf("encountered error: %+v", err)
	}
}

func TestAlibabaCloudAcrBasicAuthProvider_GetAuthTokenWithoutRegion(t *testing.T) {
	authProvider := mockAuthProvider()

	os.Setenv("ALIBABA_CLOUD_OIDC_PROVIDER_ARN", "placeholder")
	os.Setenv("ALIBABA_CLOUD_ROLE_ARN", "placeholder")
	os.Setenv("ALIBABA_CLOUD_OIDC_TOKEN_FILE", "placeholder")
	_, err := authProvider.getAcrAuthToken(testArtifactWithoutRegion)
	if err == nil {
		t.Fatalf("expected error: %+v", err)
	}

	expectedMessage := "failed to get region from image: invalid alibabacloud acr image format"
	if err.Error() != expectedMessage {
		t.Fatalf("expected message: %s, instead got error: %s", expectedMessage, err.Error())
	}
}
