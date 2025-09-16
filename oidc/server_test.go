//nolint:testpackage //whitebox testing
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"otc-auth/common"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-http-utils/headers"
	"golang.org/x/oauth2"
)

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
		idTokenVerifier    iVerifier
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
	testCtx := context.Background()

	t.Run("Success case - server starts and is shut down", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(1)
		var serverErr error

		listenerChan := make(chan net.Listener, 1)

		//nolint:unparam // `address` IS used in go func below
		mockCreateListener := func(address string, ctx context.Context) (net.Listener, error) {
			l, err := net.Listen("tcp", "localhost:0") // Use dynamic port
			if err != nil {
				return nil, err
			}
			listenerChan <- l
			return l, nil
		}

		go func() {
			defer wg.Done()
			serverErr = startAndListenHTTPServer(mockChannel, mockFlow, mockCreateListener, testCtx)
		}()

		listener := <-listenerChan
		listener.Close()
		wg.Wait()

		if serverErr == nil || !strings.Contains(serverErr.Error(), "use of closed network connection") {
			t.Errorf("expected a server closed error, but got: %v", serverErr)
		}
	})
}

func Test_flowController_Authenticate(t *testing.T) {
	expectedCreds := &common.OidcCredentialsResponse{
		BearerToken: "Bearer mock-token",
	}

	tests := []struct {
		name         string
		controller   *flowController
		authInfo     common.AuthInfo
		wantResponse *common.OidcCredentialsResponse
		wantErrMsg   string
	}{
		{
			name: "Success path",
			controller: &flowController{
				newProvider: func(ctx context.Context, issuer string) (*oidc.Provider, error) {
					return &oidc.Provider{}, nil
				},
				openURL: func(url string) error { return nil },
				startServer: func(ch chan common.OidcCredentialsResponse,
					a *authFlow, cf listenerFactory, ctx context.Context,
				) error {
					ch <- *expectedCreds
					return nil
				},
				newUUID: func() string { return "test-uuid" },
			},
			authInfo:     common.AuthInfo{IdpURL: "https://example.com"},
			wantResponse: expectedCreds,
			wantErrMsg:   "",
		},
		{
			name: "Failure when OIDC provider is unreachable",
			controller: &flowController{
				newProvider: func(ctx context.Context, issuer string) (*oidc.Provider, error) {
					return nil, errors.New("provider not found")
				},
			},
			authInfo:     common.AuthInfo{IdpURL: "https://invalid-url"},
			wantResponse: nil,
			wantErrMsg:   "provider not found",
		},
		{
			name: "Failure when local server cannot start",
			controller: &flowController{
				newProvider: func(ctx context.Context, issuer string) (*oidc.Provider, error) {
					return &oidc.Provider{}, nil
				},
				openURL: func(url string) error { return nil },
				startServer: func(ch chan common.OidcCredentialsResponse,
					a *authFlow, cf listenerFactory, ctx context.Context,
				) error {
					return errors.New("address already in use")
				},
				newUUID: func() string { return "test-uuid" },
			},
			authInfo:     common.AuthInfo{IdpURL: "https://example.com"},
			wantResponse: nil,
			wantErrMsg:   "address already in use",
		},
		{
			name: "Failure when browser fails to open",
			controller: &flowController{
				newProvider: func(ctx context.Context, issuer string) (*oidc.Provider, error) {
					return &oidc.Provider{}, nil
				},
				openURL: func(url string) error { return errors.New("unsupported OS") },
				startServer: func(ch chan common.OidcCredentialsResponse,
					a *authFlow, cf listenerFactory, ctx context.Context,
				) error {
					return nil // The server starts, but the function errors out before waiting.
				},
				newUUID: func() string { return "test-uuid" },
			},
			authInfo:     common.AuthInfo{IdpURL: "https://example.com"},
			wantResponse: nil,
			wantErrMsg:   "unsupported OS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCtx := context.Background()
			got, err := tt.controller.Authenticate(tt.authInfo, testCtx)

			if tt.wantErrMsg != "" {
				if err == nil {
					t.Fatalf("Authenticate() error = nil, wantErr %q", tt.wantErrMsg)
				}
				if err.Error() != tt.wantErrMsg {
					t.Errorf("Authenticate() error = %q, wantErrMsg %q", err.Error(), tt.wantErrMsg)
				}
			} else if err != nil {
				t.Fatalf("Authenticate() unexpected error = %v", err)
			}

			if !reflect.DeepEqual(got, tt.wantResponse) {
				t.Errorf("Authenticate() got = %v, want %v", got, tt.wantResponse)
			}
		})
	}
}

func Test_handleOIDCAuth(t *testing.T) {
	const testIDToken = "a.very.valid.jwt"

	mockOauthServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]interface{}{
			"access_token": "mock_access_token",
			idTokenField:   testIDToken,
		})
		if err != nil {
			t.Fatalf("couldn't encode token: %v", err)
		}
	}))
	t.Cleanup(mockOauthServer.Close)

	serverWithoutIDToken := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "only_this"})
		if err != nil {
			t.Fatalf("couldn't encode token: %v", err)
		}
	}))
	t.Cleanup(serverWithoutIDToken.Close)

	tests := []struct {
		name                 string
		requestURLParams     string
		mockFlowState        string
		mockOAuthConfig      oauth2.Config
		mockVerifierBehavior mockVerifier
		wantStatusCode       int
		wantBodyContains     string
		wantBearerToken      string
	}{
		{
			name:             "State mismatch returns 400",
			requestURLParams: "state=wrong-state&code=any",
			mockFlowState:    "correct-state",
			wantStatusCode:   http.StatusBadRequest,
			wantBodyContains: "state does not match",
		},
		{
			name:             "Token exchange failure returns 500",
			requestURLParams: "state=s&code=c",
			mockFlowState:    "s",
			mockOAuthConfig:  oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: "http://127.0.0.1:0/token"}}, // Unreachable
			wantStatusCode:   http.StatusInternalServerError,
			wantBodyContains: "Failed to exchange token",
		},
		{
			name:             "Token response without id_token returns 500",
			requestURLParams: "state=s&code=c",
			mockFlowState:    "s",
			mockOAuthConfig:  oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: serverWithoutIDToken.URL}},
			wantStatusCode:   http.StatusInternalServerError,
			wantBodyContains: "No id_token field",
		},
		{
			name:                 "Token verification failure returns 500",
			requestURLParams:     "state=s&code=c",
			mockFlowState:        "s",
			mockOAuthConfig:      oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
			mockVerifierBehavior: mockVerifier{ReturnError: errors.New("invalid signature")},
			wantStatusCode:       http.StatusInternalServerError,
			wantBodyContains:     "Failed to verify ID Token",
		},
		{
			name:                 "Claims extraction failure returns 500",
			requestURLParams:     "state=s&code=c",
			mockFlowState:        "s",
			mockOAuthConfig:      oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
			mockVerifierBehavior: mockVerifier{ReturnIDToken: &mockIDToken{ReturnErrorOnClaims: errors.New("malformed claims")}},
			wantStatusCode:       http.StatusInternalServerError,
			wantBodyContains:     "malformed claims",
		},
		{
			name:                 "Success returns 200 and sends on channel",
			requestURLParams:     "state=s&code=c",
			mockFlowState:        "s",
			mockOAuthConfig:      oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: mockOauthServer.URL}},
			mockVerifierBehavior: mockVerifier{ReturnIDToken: &mockIDToken{}},
			wantStatusCode:       http.StatusOK,
			wantBodyContains:     common.SuccessPageHTML,
			wantBearerToken:      fmt.Sprintf("Bearer %s", testIDToken),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			channel := make(chan common.OidcCredentialsResponse, 1)
			request := httptest.NewRequest(http.MethodGet, "/?"+tt.requestURLParams, nil)
			flow := &authFlow{
				state:           tt.mockFlowState,
				oAuth2Config:    tt.mockOAuthConfig,
				idTokenVerifier: &tt.mockVerifierBehavior,
			}

			handleOIDCAuth(recorder, request, channel, flow)

			assertHTTPResponse(t, recorder, tt.wantStatusCode, tt.wantBodyContains)
			assertChannelResponse(t, channel, tt.wantBearerToken)
		})
	}
}

func assertHTTPResponse(t *testing.T, rec *httptest.ResponseRecorder, wantCode int, wantBody string) {
	t.Helper()
	if rec.Code != wantCode {
		t.Errorf("wrong status code: got %v want %v", rec.Code, wantCode)
	}
	if !strings.Contains(rec.Body.String(), wantBody) {
		t.Errorf("body '%s' does not contain '%s'", rec.Body.String(), wantBody)
	}
}

func assertChannelResponse(t *testing.T, ch chan common.OidcCredentialsResponse, wantToken string) {
	t.Helper()

	if wantToken == "" {
		select {
		case <-ch:
			t.Error("unexpected value was sent on the channel")
		default:
			// Success: The channel is empty as expected.
		}
		return
	}

	select {
	case creds := <-ch:
		if creds.BearerToken != wantToken {
			t.Errorf("wrong bearer token: got %q want %q", creds.BearerToken, wantToken)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected a value to be sent on the channel, but received none")
	}
}
