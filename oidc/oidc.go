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
	var oidcCredentials *common.OidcCredentialsResponse
	var err error
	if authInfo.IsServiceAccount {
		oidcCredentials, err = authenticateServiceAccountWithIdp(authInfo, skipTLS, common.HTTPClientImpl{})
	} else {
		oidcCredentials, err = authenticateWithIdp(authInfo)
	}

	if err != nil {
		common.ThrowError(err)
	}

	return authenticateWithServiceProvider(*oidcCredentials, authInfo, skipTLS, common.HTTPClientImpl{})
}

//nolint:lll // This function will be removed soon
func authenticateWithServiceProvider(oidcCredentials common.OidcCredentialsResponse, authInfo common.AuthInfo, skipTLS bool, client common.HTTPClient) common.TokenResponse {
	var tokenResponse *common.TokenResponse
	url := endpoints.IdentityProviders(authInfo.IdpName, authInfo.AuthProtocol, authInfo.Region)

	request := common.GetRequest(http.MethodPost, url, nil)

	if !strings.HasPrefix(oidcCredentials.BearerToken, "Bearer ") {
		oidcCredentials.BearerToken = fmt.Sprintf("Bearer %s", oidcCredentials.BearerToken)
	}
	request.Header.Add(
		headers.Authorization, oidcCredentials.BearerToken,
	)

	response, err := client.MakeRequest(request, skipTLS) //nolint:bodyclose,lll // The body IS being closed in GetCloudCredentialsFromResponse after being read, which might be worth refactoring later
	if err != nil {
		common.ThrowError(err)
	}
	tokenResponse, err = common.GetCloudCredentialsFromResponse(response)
	if err != nil {
		common.ThrowError(err)
	}
	tokenResponse.Token.User.Name = oidcCredentials.Claims.PreferredUsername

	return *tokenResponse
}
