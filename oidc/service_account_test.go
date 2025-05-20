package oidc

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
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
			// Construct the expected http.Request manually for comparison.
			// Note: reflect.DeepEqual on http.Request is notoriously difficult and often unreliable
			// due to internal state, unexported fields, and the Body field (io.ReadCloser).
			// A more robust test might check specific fields (Method, URL, Header, Body content)
			// or use a test HTTP server. However, following the provided skeleton's structure:
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
