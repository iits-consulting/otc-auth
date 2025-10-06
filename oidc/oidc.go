package oidc

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"

	"github.com/go-http-utils/headers"
)

type AuthService struct {
	authUserFn           func(common.AuthInfo, context.Context) (*common.OidcCredentialsResponse, error)
	authServiceAccountFn func(context.Context, common.AuthInfo, common.HTTPClient) (*common.OidcCredentialsResponse, error)
	authTokenExchangeFn  func(context.Context, common.OidcCredentialsResponse,
		common.AuthInfo, common.HTTPClient) (*common.TokenResponse, error)
}

func newAuthService() *AuthService {
	return &AuthService{
		authUserFn:           authenticateWithIdp,
		authServiceAccountFn: authenticateServiceAccountWithIdp,
		authTokenExchangeFn:  authenticateWithServiceProvider,
	}
}

func (s *AuthService) authenticate(ctx context.Context,
	authInfo common.AuthInfo,
) (*common.TokenResponse, error) {
	var oidcCredentials *common.OidcCredentialsResponse
	var err error
	httpClient := common.NewHTTPClient(authInfo.SkipTLS)

	if authInfo.IsServiceAccount {
		oidcCredentials, err = s.authServiceAccountFn(ctx, authInfo, httpClient)
	} else {
		oidcCredentials, err = s.authUserFn(authInfo, ctx)
	}

	if err != nil {
		return nil, err
	}

	return s.authTokenExchangeFn(ctx, *oidcCredentials, authInfo, httpClient)
}

func AuthenticateAndGetUnscopedToken(ctx context.Context,
	authInfo common.AuthInfo,
) (*common.TokenResponse, error) {
	service := newAuthService()
	return service.authenticate(ctx, authInfo)
}

func authenticateWithServiceProvider(ctx context.Context, oidcCredentials common.OidcCredentialsResponse,
	authInfo common.AuthInfo, client common.HTTPClient,
) (*common.TokenResponse, error) {
	var tokenResponse *common.TokenResponse
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol, authInfo.Region)

	request, err := common.NewRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't get new request: %w", err)
	}

	if !strings.HasPrefix(oidcCredentials.BearerToken, "Bearer ") {
		oidcCredentials.BearerToken = fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken)
	}
	request.Header.Add(
		headers.Authorization, oidcCredentials.BearerToken,
	)

	response, err := client.MakeRequest(request)
	if err != nil {
		return nil, fmt.Errorf("couldn't make request: %w", err)
	}
	defer response.Body.Close()

	tokenResponse, err = common.GetCloudCredentialsFromResponse(response)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cloud credentials from response: %w", err)
	}
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername

	return tokenResponse, nil
}
