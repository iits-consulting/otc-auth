package oidc

import (
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	oidcCredentials := authenticateWithIdp(authInfo)

	return authenticateWithServiceProvider(oidcCredentials, authInfo)
}

func authenticateWithServiceProvider(oidcCredentials common.OidcCredentialsResponse, authInfo common.AuthInfo) (tokenResponse common.TokenResponse) {
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol)

	request := common.GetRequest(http.MethodPost, url, nil)
	request.Header.Add(headers.Authorization, oidcCredentials.BearerToken)

	response := common.HttpClientMakeRequest(request)

	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername
	defer response.Body.Close()
	return
}
