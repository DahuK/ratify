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
	"time"

	cr20181201 "github.com/alibabacloud-go/cr-20181201/v2/client"
	"github.com/alibabacloud-go/tea/tea"
)

// AcrAuthToken provides helper functions for ACR auth token data
type AcrAuthToken struct {
	AuthData map[string]*cr20181201.GetAuthorizationTokenResponseBody
}

// exists checks if authdata entries exist
func (e AcrAuthToken) exists(key string) bool {
	if len(e.AuthData) < 1 {
		return false
	}

	if _, ok := e.AuthData[key]; !ok {
		return false
	}

	if *e.AuthData[key].AuthorizationToken == "" {
		return false
	}

	return true
}

// Expiry returns the expiry time
func (a AcrAuthToken) Expiry(key string) time.Time {
	return time.UnixMilli(tea.Int64Value(a.AuthData[key].ExpireTime))
}
