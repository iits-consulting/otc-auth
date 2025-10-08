//nolint:testpackage //whitebox testing
package oidc

import (
	"context"
	"net/http"
)

type mockVerifier struct {
	ReturnError   error
	ReturnIDToken iIDToken
}

type mockIDToken struct {
	ReturnErrorOnClaims error
}

func (m *mockIDToken) Claims(v interface{}) error {
	return m.ReturnErrorOnClaims
}

func (m *mockVerifier) Verify(ctx context.Context, rawIDToken string) (iIDToken, error) {
	return m.ReturnIDToken, m.ReturnError
}

type mockHTTPClient struct {
	MakeRequestFunc func(request *http.Request) (*http.Response, error)
	Response        *http.Response
	Error           error
}

func (m mockHTTPClient) MakeRequest(request *http.Request) (*http.Response, error) {
	if m.MakeRequestFunc != nil {
		return m.MakeRequestFunc(request)
	}
	return m.Response, m.Error
}
