package oidc

import (
	"fmt"
	"net/http"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"

	"github.com/go-http-utils/headers"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo, skipTLS bool) common.TokenResponse {
	var oidcCredentials common.OidcCredentialsResponse
	if authInfo.IsServiceAccount {
		oidcCredentials = authenticateServiceAccountWithIdp(authInfo, skipTLS)
	} else {
		oidcCredentials = authenticateWithIdp(authInfo)
	}

	return authenticateWithServiceProvider(oidcCredentials, authInfo, skipTLS)
}

//nolint:lll // This function will be removed soon
func authenticateWithServiceProvider(oidcCredentials common.OidcCredentialsResponse, authInfo common.AuthInfo, skipTLS bool) common.TokenResponse {
	var tokenResponse common.TokenResponse
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol, authInfo.Region)

	request := common.GetRequest(http.MethodPost, url, nil)

	if !strings.HasPrefix(oidcCredentials.BearerToken, "Bearer ") {
		oidcCredentials.BearerToken = fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken)
	}
	request.Header.Add(
		headers.Authorization, oidcCredentials.BearerToken,
	)

	response := common.HTTPClientMakeRequest(request, skipTLS) //nolint:bodyclose,lll // The body IS being closed in GetCloudCredentialsFromResponseOrThrow after being read, which might be worth refactoring later

	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername

	return tokenResponse
}
