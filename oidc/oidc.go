package oidc

import (
	"fmt"
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	var oidcCredentials common.OidcCredentialsResponse
	if authInfo.IsServiceAccount {
		oidcCredentials = authenticateServiceAccountWithIdp(authInfo)
	} else {
		oidcCredentials = authenticateWithIdp(authInfo)
	}

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
