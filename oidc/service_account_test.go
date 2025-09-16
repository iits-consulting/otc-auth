//nolint:testpackage //whitebox testing
package oidc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"

	"otc-auth/common"
)

func Test_createServiceAccountAuthenticateRequest(t *testing.T) {
	textCtx := context.Background()
	type args struct {
		requestURL   string
		clientID     string
		clientSecret string
	}
	tests := []struct {
		name            string
		args            args
		wantURL         string
		wantBody        string
		wantContentType string
		wantUser        string
		wantPass        string
	}{
		{
			name: "basic valid request",
			args: args{
				requestURL:   "http://example.com/token",
				clientID:     "myclient",
				clientSecret: "mysecret",
			},
			wantURL:         "http://example.com/token",
			wantBody:        "grant_type=client_credentials&scope=openid",
			wantContentType: "application/x-www-form-urlencoded",
			wantUser:        "myclient",
			wantPass:        "mysecret",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := createServiceAccountAuthenticateRequest(textCtx,
				tt.args.requestURL, tt.args.clientID, tt.args.clientSecret)
			if err != nil {
				t.Errorf("couldn't create sa auth request: %v", err)
			}

			assertStringEquals(t, "URL", got.URL.String(), tt.wantURL)
			assertStringEquals(t, "Content-Type Header", got.Header.Get("Content-Type"), tt.wantContentType)
			assertRequestBody(t, got, tt.wantBody)
			assertBasicAuth(t, got, tt.wantUser, tt.wantPass)
		})
	}
}

func assertStringEquals(t *testing.T, fieldName, got, want string) {
	t.Helper() // Marks this function as a test helper. Errors will be reported from the caller's line.
	if got != want {
		t.Errorf("%s mismatch: want %q, got %q", fieldName, want, got)
	}
}

func assertRequestBody(t *testing.T, got *http.Request, wantBody string) {
	t.Helper()
	if got.Body == nil {
		t.Fatal("Request body is nil")
	}
	gotBody, err := io.ReadAll(got.Body)
	if err != nil {
		t.Fatalf("Failed to read request body: %v", err)
	}
	// Restore the body so it can be read again if needed
	got.Body = io.NopCloser(bytes.NewBuffer(gotBody))

	assertStringEquals(t, "Request Body", string(gotBody), wantBody)
}

func assertBasicAuth(t *testing.T, got *http.Request, wantUser, wantPass string) {
	t.Helper()
	gotUser, gotPass, ok := got.BasicAuth()
	if !ok {
		t.Fatal("Request is missing Basic Auth header")
	}
	assertStringEquals(t, "Basic Auth User", gotUser, wantUser)
	assertStringEquals(t, "Basic Auth Password", gotPass, wantPass)
}

type mockHTTPClient struct {
	MakeRequestFunc func(request *http.Request) (*http.Response, error)
}

func (m mockHTTPClient) MakeRequest(request *http.Request) (*http.Response, error) {
	return m.MakeRequestFunc(request)
}

func Test_authenticateServiceAccountWithIdp(t *testing.T) {
	validURL := "http://valid.idp"
	validAuth := common.AuthInfo{
		IdpURL:       validURL,
		ClientID:     "client",
		ClientSecret: "secret",
	}
	textCtx := context.Background()

	tests := []struct {
		name    string
		client  common.HTTPClient
		params  common.AuthInfo
		want    *common.OidcCredentialsResponse
		wantErr bool
	}{
		{
			name: "invalid URL",
			params: common.AuthInfo{
				IdpURL: "http://invalid url",
			},
			client:  common.HTTPClientImpl{},
			wantErr: true,
		},
		{
			name:   "HTTP request failure",
			params: validAuth,
			client: mockHTTPClient{
				MakeRequestFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusInternalServerError,
						Body:       io.NopCloser(bytes.NewBufferString("")),
					}, nil
				},
			},
			wantErr: true,
		},
		{
			name:   "body read error",
			params: validAuth,
			client: mockHTTPClient{
				MakeRequestFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(errorReader{}),
					}, nil
				},
			},
			wantErr: true,
		},
		{
			name:   "invalid JSON response",
			params: validAuth,
			client: mockHTTPClient{
				MakeRequestFunc: func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("{invalid json}")),
					}, nil
				},
			},
			wantErr: true,
		},
		{
			name:   "successful authentication",
			params: validAuth,
			client: mockHTTPClient{
				MakeRequestFunc: func(req *http.Request) (*http.Response, error) {
					if req.URL.String() != "http://valid.idp/protocol/openid-connect/token" {
						t.Errorf("Unexpected URL: %s", req.URL.String())
					}
					if req.Method != http.MethodPost {
						t.Errorf("Unexpected method: %s", req.Method)
					}

					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(`{"id_token":"test-token"}`)),
					}, nil
				},
			},
			want: &common.OidcCredentialsResponse{
				BearerToken: "test-token",
				Claims: struct {
					PreferredUsername string `json:"preferred_username"`
				}(struct {
					PreferredUsername string
				}{PreferredUsername: "ServiceAccount"}),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.client
			if client == nil {
				client = common.HTTPClientImpl{}
			}

			got, err := authenticateServiceAccountWithIdp(textCtx, tt.params, client)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got = %v, want %v", got, tt.want)
			}
		})
	}
}

type errorReader struct{}

func (errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("simulated read error")
}
