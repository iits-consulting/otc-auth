package oidc

import (
	"fmt"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"strings"

	"github.com/go-http-utils/headers"
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

	if !strings.HasPrefix(oidcCredentials.BearerToken, "Bearer ") {
		oidcCredentials.BearerToken = fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken)
	}
	request.Header.Add(
		headers.Authorization, oidcCredentials.BearerToken,
	)

	response := common.HttpClientMakeRequest(request)

	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername
	defer response.Body.Close()
	return
}
