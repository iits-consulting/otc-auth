package oidc

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"otc-auth/common"
	"reflect"
	"testing"
)

func Test_createServiceAccountAuthenticateRequest(t *testing.T) {
	type args struct {
		requestURL   string
		clientID     string
		clientSecret string
	}
	tests := []struct {
		name string
		args args
		want *http.Request
	}{
		{
			name: "basic valid request",
			args: args{
				requestURL:   "http://example.com/token",
				clientID:     "myclient",
				clientSecret: "mysecret",
			},
			want: func() *http.Request {
				expectedURL := "http://example.com/token"
				data := url.Values{}
				data.Set("grant_type", "client_credentials")
				data.Set("scope", "openid")
				expectedBodyContent := data.Encode()

				bodyReader := io.NopCloser(bytes.NewReader([]byte(expectedBodyContent)))

				req, err := http.NewRequest(http.MethodPost, expectedURL, bodyReader)
				if err != nil {
					t.Fatalf("Failed to create expected request: %v", err)
				}

				req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth("myclient", "mysecret") // TODO - consts

				return req
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createServiceAccountAuthenticateRequest(tt.args.requestURL, tt.args.clientID, tt.args.clientSecret)

			wantBody, err := io.ReadAll(tt.want.Body)
			if err != nil {
				t.Errorf("error reading want body: %v", err)
			}
			gotBody, err := io.ReadAll(got.Body)
			if err != nil {
				t.Errorf("error reading got body: %v", err)
			}
			if string(wantBody) != string(gotBody) {
				t.Errorf("body mismatch -> want: %s, got: %s", string(wantBody), string(gotBody))
			}

			if got.URL.String() != tt.want.URL.String() {
				t.Errorf("url mismatch -> want: %s, got: %s", tt.want.URL.String(), got.URL.String())
			}

			if got.Header.Get("Content-Type") != tt.want.Header.Get("Content-Type") {
				t.Errorf("Content-Type header mismatch -> want: %s, got: %s", tt.want.Header.Get("Content-Type"), got.Header.Get("Content-Type"))
			}

			gotUser, gotPass, _ := got.BasicAuth()
			if gotUser != tt.args.clientID {
				t.Errorf("basicauth user mismatch -> want: %s, got: %s", tt.args.clientID, gotUser)
			}

			if gotPass != tt.args.clientSecret {
				t.Errorf("basicauth password mismatch -> want: %s, got: %s", tt.args.clientSecret, gotPass)
			}
		})
	}
}

type mockHTTPClient struct {
	MakeRequestFunc func(request *http.Request, skipTLS bool) (*http.Response, error)
}

func (m mockHTTPClient) MakeRequest(request *http.Request, skipTLS bool) (*http.Response, error) {
	return m.MakeRequestFunc(request, skipTLS)
}

func Test_authenticateServiceAccountWithIdp(t *testing.T) {
	validURL := "http://valid.idp"
	validAuth := common.AuthInfo{
		IdpURL:       validURL,
		ClientID:     "client",
		ClientSecret: "secret",
	}

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
				MakeRequestFunc: func(req *http.Request, skipTLS bool) (*http.Response, error) {
					return &http.Response{
						StatusCode: 500,
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
				MakeRequestFunc: func(req *http.Request, skipTLS bool) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(errorReader{}),
					}, nil
				}},
			wantErr: true,
		},
		{
			name:   "invalid JSON response",
			params: validAuth,
			client: mockHTTPClient{
				MakeRequestFunc: func(req *http.Request, skipTLS bool) (*http.Response, error) {
					return &http.Response{
						StatusCode: 200,
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
				MakeRequestFunc: func(req *http.Request, skipTLS bool) (*http.Response, error) {
					if req.URL.String() != "http://valid.idp/protocol/openid-connect/token" {
						t.Errorf("Unexpected URL: %s", req.URL.String())
					}
					if req.Method != "POST" {
						t.Errorf("Unexpected method: %s", req.Method)
					}

					// Return valid response
					return &http.Response{
						StatusCode: 200,
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

			got, err := authenticateServiceAccountWithIdp(tt.params, false, client)
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
