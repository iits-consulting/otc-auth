package oidc

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"

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

//nolint:lll // This function will be removed soon
func authenticateWithServiceProvider(oidcCredentials common.OidcCredentialsResponse, authInfo common.AuthInfo) common.TokenResponse {
	var tokenResponse common.TokenResponse
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol, authInfo.Region)

	request := common.GetRequest(http.MethodPost, url, nil)

	if !strings.HasPrefix(oidcCredentials.BearerToken, "Bearer ") {
		oidcCredentials.BearerToken = fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken)
	}
	request.Header.Add(
		headers.Authorization, oidcCredentials.BearerToken,
	)

	response := common.HTTPClientMakeRequest(request) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon

	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			common.OutputErrorToConsoleAndExit(err)
		}
	}(response.Body)
	return tokenResponse
}
