package oidc

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"otc-auth/common"
	"strings"
	"sync"
	"testing"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-http-utils/headers"
	"golang.org/x/oauth2"
)

// mockVerifier is a test implementation of the IVerifier interface.
type mockVerifier struct {
	ReturnError   error
	ReturnIDToken *oidc.IDToken
}

// Verify implements the IVerifier interface for our mock.
func (m *mockVerifier) Verify(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	return m.ReturnIDToken, m.ReturnError
}

func Test_authFlow_handleRoot(t *testing.T) {
	// Common setup for our tests
	const testState = "test-state-123"
	const testClientID = "my-test-client"

	// This is the URL that the oauth2 library will correctly generate.
	const testRedirectURL = "https://example.com/auth?client_id=" + testClientID + "&response_type=code&state=" + testState

	commonOauthConfig := oauth2.Config{
		ClientID: testClientID, // Provide the ClientID
		Endpoint: oauth2.Endpoint{AuthURL: "https://example.com/auth"},
	}

	tests := []struct {
		name               string
		request            *http.Request
		idTokenVerifier    IVerifier // Use our interface here
		expectedStatusCode int
		expectedLocation   string // To check for redirects
	}{
		{
			name:               "No Authorization Header should redirect",
			request:            httptest.NewRequest(http.MethodGet, "/", nil),
			idTokenVerifier:    &mockVerifier{}, // Doesn't matter for this case
			expectedStatusCode: http.StatusFound,
			expectedLocation:   testRedirectURL,
		},
		{
			name: "Malformed Authorization Header should return 400 Bad Request",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(headers.Authorization, "BearerTokenWithoutSpace") // Malformed
				return req
			}(),
			idTokenVerifier:    &mockVerifier{},
			expectedStatusCode: http.StatusBadRequest,
			expectedLocation:   "", // No redirect expected
		},
		{
			name: "Invalid Token should redirect",
			request: func() *http.Request {
				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.Header.Set(headers.Authorization, "Bearer invalid-token")
				return req
			}(),
			idTokenVerifier: &mockVerifier{
				ReturnError:   errors.New("oidc: token is invalid"), // Simulate a verification error
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
				ReturnError:   nil,             // Simulate successful verification
				ReturnIDToken: &oidc.IDToken{}, // Return a non-nil token
			},
			expectedStatusCode: http.StatusOK,
			expectedLocation:   "", // No redirect expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup the authFlow with the verifier for this specific test case
			a := &authFlow{
				oAuth2Config:    commonOauthConfig,
				idTokenVerifier: tt.idTokenVerifier,
				state:           testState,
			}

			// Use httptest.NewRecorder to capture the response
			recorder := httptest.NewRecorder()

			// Call the handler
			a.handleRoot(recorder, tt.request)

			// Assert the status code
			if recorder.Code != tt.expectedStatusCode {
				t.Errorf("handler returned wrong status code: got %v want %v", recorder.Code, tt.expectedStatusCode)
			}

			// Assert the Location header for redirects
			if tt.expectedLocation != "" {
				location := recorder.Header().Get("Location")
				// The AuthCodeURL function might add extra parameters, so we check if it starts with our expected URL
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

	t.Run("Failure case - listener cannot be created", func(t *testing.T) {
		expectedErr := errors.New("failed to listen on port")
		mockCreateListener := func(address string) (net.Listener, error) {
			return nil, expectedErr
		}

		err := startAndListenHTTPServer(mockChannel, mockFlow, mockCreateListener)

		if !errors.Is(err, expectedErr) {
			t.Errorf("expected error '%v', but got: %v", expectedErr, err)
		}
	})
}
