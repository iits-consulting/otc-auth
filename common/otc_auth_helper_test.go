package common_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"otc-auth/common"
	"otc-auth/common/xheaders"

	"github.com/google/go-cmp/cmp"
)

func mockResponse(status string, statusCode int, headers map[string]string, body string) *http.Response {
	r := &http.Response{
		Status:     status,
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	return r
}

func TestGetCloudCredentialsFromResponse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		response   *http.Response
		want       *common.TokenResponse
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "success with token in header",
			response: mockResponse("200 OK", 200,
				map[string]string{xheaders.XSubjectToken: "test-token"},
				`{"token": {"expires_at": "2023-01-01", "issued_at": "2023-01-01", "user": {"domain": {"id": "domain-id", "name": "domain-name"}, "name": "user-name"}}}`), //nolint:lll // We expect this format
			want: &common.TokenResponse{
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
				}(struct {
					Secret    string `json:"-"`
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
					Secret:    "test-token",
					ExpiresAt: "2023-01-01",
					IssuedAt:  "2023-01-01",
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
							ID:   "domain-id",
							Name: "domain-name",
						},
						Name: "user-name",
					},
				}),
			},
			wantErr: false,
		},
		{
			name: "error when no token in header and MFA failure",
			response: mockResponse("403 Forbidden", 403, nil,
				`{"error": "mfa totp code verify fail"}`),
			wantErr:    true,
			wantErrMsg: `{"error": "mfa totp code verify fail"}`,
		},
		{
			name: "error when no token in header and other error",
			response: mockResponse("401 Unauthorized", 401, nil,
				`{"error": "auth failed"}`),
			wantErr:    true,
			wantErrMsg: `{"error": "auth failed"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := common.GetCloudCredentialsFromResponse(tt.response)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("GetCloudCredentialsFromResponse() error = nil, wantErr %v", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErrMsg) {
					t.Errorf("GetCloudCredentialsFromResponse() error = %q, want contains %q", err.Error(), tt.wantErrMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("GetCloudCredentialsFromResponse() unexpected error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("GetCloudCredentialsFromResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
