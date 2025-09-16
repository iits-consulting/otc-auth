package oidc

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/xheaders"
	"reflect"
	"strings"
	"testing"
)

func Test_authenticateWithServiceProvider(t *testing.T) {
	ctx := context.Background()
	authInfo := common.AuthInfo{IdpName: "myidp", AuthProtocol: "oidc", Region: "test-region"}

	expectedToken := &common.TokenResponse{
		Token: struct {
			Secret    string
			ExpiresAt string `json:"expires_at"`
			IssuedAt  string `json:"issued_at"`
			User      struct {
				Domain struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"domain"`
				Name string `json:"name"`
			} `json:"user"`
		}{
			Secret:    "a-very-long-secret",
			ExpiresAt: "2025-01-01T00:00:00Z",
			IssuedAt:  "2024-12-31T23:00:00Z",
			User: struct {
				Domain struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"domain"`
				Name string `json:"name"`
			}{
				Domain: struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				}{
					ID:   "domain123",
					Name: "Default",
				},
				Name: "test-user",
			},
		},
	}

	successBody, err := json.Marshal(expectedToken)
	if err != nil {
		t.Fatalf("Failed to marshal success body: %v", err)
	}

	tests := []struct {
		name            string
		oidcCredentials common.OidcCredentialsResponse
		mockClient      common.HTTPClient
		want            *common.TokenResponse
		wantErrMsg      string
	}{
		{
			name: "Success path with full Bearer token",
			oidcCredentials: common.OidcCredentialsResponse{
				BearerToken: "Bearer real-token",
				Claims: struct {
					PreferredUsername string `json:"preferred_username"`
				}(struct {
					PreferredUsername string
				}{PreferredUsername: "test-user"}),
			},
			mockClient: &mockHTTPClient{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
					Header: http.Header{
						xheaders.XSubjectToken: []string{"a-very-long-secret"},
					},
					Body: io.NopCloser(bytes.NewReader(successBody)),
				},
				Error: nil,
			},
			want:       expectedToken,
			wantErrMsg: "",
		},
		{
			name: "Success path with token needing 'Bearer ' prefix",
			oidcCredentials: common.OidcCredentialsResponse{
				BearerToken: "raw-token-no-prefix",
				Claims: struct {
					PreferredUsername string `json:"preferred_username"`
				}(struct {
					PreferredUsername string
				}{PreferredUsername: "test-user"}),
			},
			mockClient: &mockHTTPClient{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
					Header: http.Header{
						xheaders.XSubjectToken: []string{"a-very-long-secret"},
					},
					Body: io.NopCloser(bytes.NewReader(successBody)),
				},
				Error: nil,
			},
			want:       expectedToken,
			wantErrMsg: "",
		},
		{
			name:            "Failure when client.MakeRequest fails",
			oidcCredentials: common.OidcCredentialsResponse{BearerToken: "any-token"},
			mockClient: &mockHTTPClient{
				Response: nil,
				Error:    errors.New("network connection refused"),
			},
			want:       nil,
			wantErrMsg: "couldn't make request: network connection refused",
		},
		{
			name:            "Failure when response body is unparsable",
			oidcCredentials: common.OidcCredentialsResponse{BearerToken: "any-token"},
			mockClient: &mockHTTPClient{
				Response: &http.Response{
					StatusCode: http.StatusOK,
					Status:     fmt.Sprintf("%d %s", http.StatusOK, http.StatusText(http.StatusOK)),
					Header: http.Header{
						xheaders.XSubjectToken: []string{"the-real-unscoped-token"},
					},
					Body: io.NopCloser(strings.NewReader("{not-valid-json]")),
				},
				Error: nil,
			},
			want:       nil,
			wantErrMsg: "couldn't get cloud credentials from response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, authErr := authenticateWithServiceProvider(ctx, tt.oidcCredentials, authInfo, tt.mockClient)

			if tt.wantErrMsg != "" {
				if authErr == nil {
					t.Fatalf("authenticateWithServiceProvider() error = nil, wantErr %q", tt.wantErrMsg)
				}

				if !strings.Contains(authErr.Error(), tt.wantErrMsg) {
					t.Errorf("authenticateWithServiceProvider() error = %q, wantErrMsg to contain %q", authErr.Error(), tt.wantErrMsg)
				}
			} else if authErr != nil {
				t.Fatalf("authenticateWithServiceProvider() unexpected error = %v", authErr)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("authenticateWithServiceProvider() got = %v, want %v", got, tt.want)
			}
		})
	}
}
