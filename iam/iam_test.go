//nolint:testpackage // whitebox testing
package iam

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"otc-auth/config"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
)

// TestExtractToken_ReadsSubjectTokenHeader is a canary for the one SDK
// extraction iam depends on: tokens.Create(...).ExtractToken().ID reads the
// X-Subject-Token response header. If it breaks, verify where the SDK now
// reads the token ID.
func TestExtractToken_ReadsSubjectTokenHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Subject-Token", "the-token-id")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"token":{"expires_at":"2022-11-30T14:01:54.956000Z"}}`))
	}))
	t.Cleanup(server.Close)

	client := &golangsdk.ServiceClient{
		ProviderClient: &golangsdk.ProviderClient{},
		Endpoint:       server.URL + "/",
	}

	authOpts := &golangsdk.AuthOptions{
		DomainName: "domain",
		Username:   "user",
		Password:   "pass",
	}
	token, err := tokens.Create(client, authOpts).ExtractToken()
	if err != nil {
		t.Fatalf("ExtractToken: %v", err)
	}
	if token.ID != "the-token-id" {
		t.Errorf("ID = %q, want %q — SDK may have changed where the token ID is read",
			token.ID, "the-token-id")
	}
}

func Test_buildUnscopedTokenResponse(t *testing.T) {
	t.Parallel()

	//nolint:lll // single-line JSON fixture
	body := []byte(`{"token":{"expires_at":"2022-11-30T14:01:54.956000Z","issued_at":"2022-11-29T14:01:54.956000Z","user":{"domain":{"id":"domain-id","name":"domain-name"},"name":"user-name"}}}`)

	got, err := buildUnscopedTokenResponse(body, "secret-token-id")
	if err != nil {
		t.Fatalf("buildUnscopedTokenResponse() error = %v", err)
	}
	// Secret has no JSON tag; it must come from the extracted token ID, not the body.
	if got.Token.Secret != "secret-token-id" {
		t.Errorf("Secret = %q, want %q", got.Token.Secret, "secret-token-id")
	}
	if got.Token.ExpiresAt != "2022-11-30T14:01:54.956000Z" {
		t.Errorf("ExpiresAt = %q, want %q", got.Token.ExpiresAt, "2022-11-30T14:01:54.956000Z")
	}
	if got.Token.User.Name != "user-name" {
		t.Errorf("User.Name = %q, want %q", got.Token.User.Name, "user-name")
	}

	if _, errInvalid := buildUnscopedTokenResponse([]byte("{not json"), "x"); errInvalid == nil {
		t.Error("expected error for invalid JSON body, got nil")
	}
}

type mockTokenCreator struct {
	tokenToReturn *tokens.Token
	errorToReturn error
}

func (m *mockTokenCreator) CreateToken(opts golangsdk.AuthOptions) (*tokens.Token, error) {
	return m.tokenToReturn, m.errorToReturn
}

type mockConfigStore struct {
	cloudToReturn   *config.Cloud
	getError        error
	saveError       error
	SaveCalled      bool
	SavedCloudState *config.Cloud
	GetCallCount    int
}

func (m *mockConfigStore) GetActiveCloud() (*config.Cloud, error) {
	m.GetCallCount++
	return m.cloudToReturn, m.getError
}

func (m *mockConfigStore) SaveActiveCloud(cloud config.Cloud) error {
	if m.saveError == nil {
		m.SaveCalled = true
		m.SavedCloudState = &cloud
	}
	return m.saveError
}

func TestGetScopedToken(t *testing.T) {
	validConfigToken := &config.Token{
		Secret:    "new-secret",
		ExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}
	expiredConfigToken := &config.Token{
		Secret:    "new-secret",
		ExpiresAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
	}
	refreshedConfigToken := &config.Token{
		Secret:    "new-secret",
		ExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
	}

	refreshedGopherToken, err := configTokenToGopherToken(refreshedConfigToken)
	if err != nil {
		t.Errorf("couldn't convert config token to gopher token: %v", err)
	}

	tests := []struct {
		name             string
		projectName      string
		setupMockStore   func() ConfigStore
		mockTokenCreator TokenCreator
		want             *config.Token
		wantErr          bool
		wantSaveCalled   bool
	}{
		{
			name:        "Success - Returns cached token when valid",
			projectName: "p1",
			setupMockStore: func() ConfigStore {
				configWithValidToken := &config.Cloud{Projects: config.Projects{{
					NameAndIDResource: config.NameAndIDResource{
						Name: "p1",
						ID:   "id1",
					},
					ScopedToken: *validConfigToken,
				}}, Region: "eu-de"}
				return &mockConfigStore{cloudToReturn: configWithValidToken}
			},
			mockTokenCreator: &mockTokenCreator{}, // Not used in this path
			want:             validConfigToken,
			wantErr:          false,
			wantSaveCalled:   false,
		},
		{
			name:        "Success - Refreshes token when expired",
			projectName: "p1",
			setupMockStore: func() ConfigStore {
				configWithExpiredToken := &config.Cloud{Projects: config.Projects{{
					NameAndIDResource: config.NameAndIDResource{
						Name: "p1",
						ID:   "id1",
					},
					ScopedToken: *expiredConfigToken,
				}}, Region: "eu-de"}
				return &mockConfigStore{cloudToReturn: configWithExpiredToken}
			},
			mockTokenCreator: &mockTokenCreator{tokenToReturn: refreshedGopherToken},
			want:             refreshedConfigToken,
			wantErr:          false,
			wantSaveCalled:   true,
		},
		{
			name:        "Failure - GetActiveCloud fails",
			projectName: "p1",
			setupMockStore: func() ConfigStore {
				return &mockConfigStore{getError: fmt.Errorf("disk error")}
			},
			mockTokenCreator: &mockTokenCreator{},
			wantErr:          true,
			wantSaveCalled:   false,
		},
		{
			name:        "Failure - Project not found in active cloud",
			projectName: "missing",
			setupMockStore: func() ConfigStore {
				configWithOtherProject := &config.Cloud{Projects: config.Projects{{
					NameAndIDResource: config.NameAndIDResource{
						Name: "p1",
						ID:   "id1",
					},
					ScopedToken: *validConfigToken,
				}}, Region: "eu-de"}
				return &mockConfigStore{cloudToReturn: configWithOtherProject}
			},
			mockTokenCreator: &mockTokenCreator{},
			wantErr:          true,
			wantSaveCalled:   false,
		},
		{
			name:        "Failure - Token refresh fails",
			projectName: "p1",
			setupMockStore: func() ConfigStore {
				configWithExpiredToken := &config.Cloud{Projects: config.Projects{{
					NameAndIDResource: config.NameAndIDResource{
						Name: "p1",
						ID:   "id1",
					},
					ScopedToken: *expiredConfigToken,
				}}, Region: "eu-de"}
				return &mockConfigStore{cloudToReturn: configWithExpiredToken}
			},
			mockTokenCreator: &mockTokenCreator{errorToReturn: fmt.Errorf("api error")},
			wantErr:          true,
			wantSaveCalled:   false,
		},
		{
			name:        "Failure - Config save fails",
			projectName: "p1",
			setupMockStore: func() ConfigStore {
				configWithExpiredToken := &config.Cloud{Projects: config.Projects{{
					NameAndIDResource: config.NameAndIDResource{
						Name: "p1",
						ID:   "id1",
					},
					ScopedToken: *expiredConfigToken,
				}}, Region: "eu-de"}
				return &mockConfigStore{cloudToReturn: configWithExpiredToken, saveError: fmt.Errorf("disk full")}
			},
			mockTokenCreator: &mockTokenCreator{tokenToReturn: refreshedGopherToken},
			wantErr:          true,
			wantSaveCalled:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := tt.setupMockStore()
			got, errToken := GetScopedToken(mockStore, tt.mockTokenCreator, tt.projectName)
			if (errToken != nil) != tt.wantErr {
				t.Errorf("GetScopedToken() error = %v, wantErr %v", errToken, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetScopedToken() got = %v, want %v", got, tt.want)
			}
			if store, ok := mockStore.(*mockConfigStore); ok {
				if store.SaveCalled != tt.wantSaveCalled {
					t.Errorf("Expected SaveCalled to be %v, but got %v", tt.wantSaveCalled, store.SaveCalled)
				}
			}
		})
	}
}

func Test_gopherTokenToConfigToken(t *testing.T) {
	testTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)
	expectedTimeString := "2023-10-27T10:00:00Z"

	tests := []struct {
		name        string
		gopherToken *tokens.Token
		want        *config.Token
		wantErr     bool
	}{
		{
			name: "Success - Converts a valid token",
			gopherToken: &tokens.Token{
				ID:        "test-id",
				ExpiresAt: testTime,
			},
			want: &config.Token{
				Secret:    "test-id",
				ExpiresAt: expectedTimeString,
			},
			wantErr: false,
		},
		{
			name:        "Failure - Handles nil input gracefully",
			gopherToken: nil,
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, errConv := gopherTokenToConfigToken(tt.gopherToken)
			if (errConv != nil) != tt.wantErr {
				t.Errorf("error converting token: %v", errConv)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("gopherTokenToConfigToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_configTokenToGopherToken(t *testing.T) {
	testTime := time.Date(2023, 10, 27, 10, 0, 0, 0, time.UTC)
	validTimeString := "2023-10-27T10:00:00Z"

	tests := []struct {
		name        string
		configToken *config.Token
		want        *tokens.Token
		wantErr     bool
	}{
		{
			name: "Success - Converts a valid token",
			configToken: &config.Token{
				Secret:    "test-id",
				ExpiresAt: validTimeString,
			},
			want: &tokens.Token{
				ID:        "test-id",
				ExpiresAt: testTime,
			},
			wantErr: false,
		},
		{
			name: "Failure - Returns error on invalid time format",
			configToken: &config.Token{
				Secret:    "test-id",
				ExpiresAt: "not-a-valid-time",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name:        "Failure - Returns error on nil input",
			configToken: nil,
			want:        nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configTokenToGopherToken(tt.configToken)

			if (err != nil) != tt.wantErr {
				t.Errorf("configTokenToGopherToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("configTokenToGopherToken() = %v, want %v", got, tt.want)
			}
		})
	}
}
