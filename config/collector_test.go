// Copyright The Perses Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/perses/perses/pkg/model/api/v1/secret"
	"github.com/perses/spec/go/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTPClientOAuthClientSecretFile(t *testing.T) {
	var receivedClientSecret string
	tokenServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, receivedClientSecret, _ = r.BasicAuth()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"token","token_type":"Bearer"}`))
	}))
	defer tokenServer.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	secretFile := filepath.Join(t.TempDir(), "client_secret")
	require.NoError(t, os.WriteFile(secretFile, []byte("secret-from-file"), 0600))

	client, err := NewHTTPClient(HTTPClient{
		URL: common.MustParseURL(server.URL),
		OAuth: &secret.OAuth{
			ClientID:         "client",
			ClientSecretFile: secretFile,
			TokenURL:         tokenServer.URL,
			// Send the client credentials with the HTTP basic authorization.
			AuthStyle: 2,
		},
	})
	require.NoError(t, err)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	_ = resp.Body.Close()
	assert.Equal(t, "secret-from-file", receivedClientSecret)
}

func TestNewHTTPClientOAuthClientSecretFileMissing(t *testing.T) {
	_, err := NewHTTPClient(HTTPClient{
		URL: common.MustParseURL("http://localhost"),
		OAuth: &secret.OAuth{
			ClientID:         "client",
			ClientSecretFile: filepath.Join(t.TempDir(), "missing"),
			TokenURL:         "http://localhost/token",
		},
	})
	assert.Error(t, err)
}
