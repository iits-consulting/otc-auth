package oidc

import (
	"fmt"
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	oidcCredentials := authenticateServiceAccountWithIdp(authInfo) // authenticateServiceAccountWithIdp(authInfo)

	return authenticateWithServiceProvider(oidcCredentials, authInfo)
}

func authenticateWithServiceProvider(oidcCredentials common.OidcCredentialsResponse, authInfo common.AuthInfo) (tokenResponse common.TokenResponse) {
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol)

	request := common.GetRequest(http.MethodPost, url, nil)
	request.Header.Add(
		headers.Authorization, fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken),
	)

	response := common.HttpClientMakeRequest(request)

	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername
	defer response.Body.Close()
	return
}
