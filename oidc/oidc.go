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

func AuthenticateAndGetUnscopedToken(ctx context.Context, authInfo common.AuthInfo,
	skipTLS bool,
) (*common.TokenResponse, error) {
	var oidcCredentials *common.OidcCredentialsResponse
	var err error
	authCtx := context.Background()
	httpClient := common.NewHTTPClient(skipTLS)
	if authInfo.IsServiceAccount {
		oidcCredentials, err = authenticateServiceAccountWithIdp(ctx, authInfo, httpClient)
	} else {
		oidcCredentials, err = authenticateWithIdp(authInfo, authCtx)
	}

	if err != nil {
		common.ThrowError(err)
	}

	return authenticateWithServiceProvider(ctx, *oidcCredentials, authInfo, httpClient)
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
