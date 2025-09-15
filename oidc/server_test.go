package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"otc-auth/common"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-http-utils/headers"
	"golang.org/x/oauth2"
)

type mockVerifier struct {
	ReturnError   error
	ReturnIDToken IIDToken
}

type mockIDToken struct {
	ReturnErrorOnClaims error
}

func (m *mockIDToken) Claims(v interface{}) error {
	return m.ReturnErrorOnClaims
}

func (m *mockVerifier) Verify(ctx context.Context, rawIDToken string) (IIDToken, error) {
	return m.ReturnIDToken, m.ReturnError
}

func Test_authFlow_handleRoot(t *testing.T) {
	const testState = "test-state-123"
	const testClientID = "my-test-client"

	const testRedirectURL = "https://example.com/auth?client_id=" + testClientID + "&response_type=code&state=" + testState

	commonOauthConfig := oauth2.Config{
		ClientID: testClientID, // Provide the ClientID
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.com/auth"},
	}

	tests := []struct {
		name               string
		request            *http.Request
		idTokenVerifier    IVerifier
		expectedStatusCode int
		expectedLocation   string
	}{
		{
			name:               "No Authorization Header should redirect",
			request:            httptest.NewRequest(http.MethodGet, "/", nil),
			idTokenVerifier:    &mockVerifier{},
			expectedStatusCode: http.StatusFound,
			expectedLocation:   testRedirectURL,
		},
		{
			name: "Malformed Authorization Header should return 400 Bad Request",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(headers.Authorization, "BearerTokenWithoutSpace")
				return req
			}(),
			idTokenVerifier:    &mockVerifier{},
			expectedStatusCode: http.StatusBadRequest,
			expectedLocation:   "",
		},
		{
			name: "Invalid Token should redirect",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(headers.Authorization, "Bearer invalid-token")
				return req
			}(),
			idTokenVerifier: &mockVerifier{
				ReturnError:   errors.New("oidc: token is invalid"),
				ReturnIDToken: nil,
			},
			expectedStatusCode: http.StatusFound,
			expectedLocation:   testRedirectURL,
		},
		{
			name: "Valid Token should return 200 OK",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(headers.Authorization, "Bearer valid-token")
				return req
			}(),
			idTokenVerifier: &mockVerifier{
				ReturnError:   nil,
				ReturnIDToken: &oidc.IDToken{},
			},
			expectedStatusCode: http.StatusOK,
			expectedLocation:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &authFlow{
				oAuth2Config:    commonOauthConfig,
				idTokenVerifier: tt.idTokenVerifier,
				state:           testState,
			}

			recorder := httptest.NewRecorder()

			a.handleRoot(recorder, tt.request)

			if recorder.Code != tt.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, tt.expectedStatusCode)
			}

			if tt.expectedLocation != "" {
				location := recorder.Header().Get("Location")
				if !strings.HasPrefix(location, tt.expectedLocation) {
					t.Errorf("handler returned wrong redirect location: got %v want prefix %v", location, tt.expectedLocation)
				}
			}
		})
	}
}

func Test_startAndListenHTTPServer(t *testing.T) {
	mockFlow := &authFlow{}
	mockChannel := make(chan common.OidcCredentialsResponse)

	t.Run("Success case - server starts and is shut down", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		var serverErr error

		listenerChan := make(chan net.Listener, 1)
		mockCreateListener := func(address string) (net.Listener, error) {
			l, err := net.Listen("tcp", "localhost:0") // Use dynamic port
			if err != nil {
				return nil, err
			}
			listenerChan <- l
			return l, nil
		}

		go func() {
			defer wg.Done()
			serverErr = startAndListenHTTPServer(mockChannel, mockFlow, mockCreateListener)
		}()

		listener := <-listenerChan
		listener.Close()
		wg.Wait()

		if serverErr == nil || !strings.Contains(serverErr.Error(), "use of closed network connection") {
			t.Errorf("expected a server closed error, but got: %v", serverErr)
		}
	})
}

func Test_handleOIDCAuth(t *testing.T) {
	const testState = "state-abc-123"
	const testCode = "code-xyz-789"
	const testIDToken = "a.very.valid.jwt"

	verifier := &mockVerifier{}

	mockOauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock_access_token",
			"token_type":   "Bearer",
			"expires_in":   3600,
			idTokenField:   testIDToken,
		})
	}))
	t.Cleanup(mockOauthServer.Close)

	tests := []struct {
		name                 string
		request              *http.Request
		authFlow             *authFlow
		setupMocks           func()
		expectedStatusCode   int
		expectedBodyContains string
		expectChannelSend    bool
	}{
		{
			name:                 "State mismatch should return 400 Bad Request",
			request:              httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=wrong-state&code=%s", testCode), nil),
			authFlow:             &authFlow{state: testState},
			setupMocks:           func() {},
			expectedStatusCode:   http.StatusBadRequest,
			expectedBodyContains: "state does not match",
		},
		{
			name:    "Token exchange failure should return 500",
			request: httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=%s&code=bad-code", testState), nil),
			authFlow: &authFlow{
				state: testState,
				oAuth2Config: oauth2.Config{
					Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:0/token"},
				},
			},
			setupMocks:           func() {},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "Failed to exchange token",
		},
		{
			name: "Token response without id_token field should return 500",
			authFlow: &authFlow{
				state:           testState,
				idTokenVerifier: verifier,
				oAuth2Config: func() oauth2.Config {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "only_this"})
					}))
					t.Cleanup(server.Close)
					return oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: server.URL}}
				}(),
			},
			request:              httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=%s&code=%s", testState, testCode), nil),
			setupMocks:           func() {},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "No id_token field",
		},
		{
			name:    "Token verification failure should return 500",
			request: httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=%s&code=%s", testState, testCode), nil),
			authFlow: &authFlow{
				state:           testState,
				oAuth2Config:    oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
				idTokenVerifier: verifier,
			},
			setupMocks: func() {
				verifier.ReturnError = errors.New("invalid signature")
				verifier.ReturnIDToken = nil
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "Failed to verify ID Token",
		},
		{
			name:    "Claims extraction failure should return 500",
			request: httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=%s&code=%s", testState, testCode), nil),
			authFlow: &authFlow{
				state:           testState,
				oAuth2Config:    oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
				idTokenVerifier: verifier,
			},
			setupMocks: func() {
				verifier.ReturnError = nil
				verifier.ReturnIDToken = &mockIDToken{ReturnErrorOnClaims: errors.New("malformed claims")}
			},
			expectedStatusCode:   http.StatusInternalServerError,
			expectedBodyContains: "malformed claims",
		},
		{
			name:    "Successful authentication should return 200 OK and send on channel",
			request: httptest.NewRequest(http.MethodGet, fmt.Sprintf("/?state=%s&code=%s", testState, testCode), nil),
			authFlow: &authFlow{
				state:           testState,
				oAuth2Config:    oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
				idTokenVerifier: verifier,
			},
			setupMocks: func() {
				verifier.ReturnError = nil
				verifier.ReturnIDToken = &mockIDToken{ReturnErrorOnClaims: nil}
			},
			expectedStatusCode:   http.StatusOK,
			expectedBodyContains: common.SuccessPageHTML,
			expectChannelSend:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			channel := make(chan common.OidcCredentialsResponse, 1)
			recorder := httptest.NewRecorder()

			tt.setupMocks()

			handleOIDCAuth(recorder, tt.request, channel, tt.authFlow)

			if recorder.Code != tt.expectedStatusCode {
				t.Errorf("wrong status code: got %v want %v", recorder.Code, tt.expectedStatusCode)
			}
			if !strings.Contains(recorder.Body.String(), tt.expectedBodyContains) {
				t.Errorf("body '%s' does not contain '%s'", recorder.Body.String(), tt.expectedBodyContains)
			}

			if tt.expectChannelSend {
				select {
				case creds := <-channel:
					expectedBearer := fmt.Sprintf("Bearer %s", testIDToken)
					if creds.BearerToken != expectedBearer {
						t.Errorf("wrong bearer token: got %s want %s", creds.BearerToken, expectedBearer)
					}
				case <-time.After(100 * time.Millisecond):
					t.Error("expected a value to be sent on the channel, but received none")
				}
			}
		})
	}
}
