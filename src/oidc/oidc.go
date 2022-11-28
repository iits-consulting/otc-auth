package oidc

import (
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
	"strings"
)

func AuthenticateAndGetUnscopedToken(params common.AuthInfo) (unscopedToken string, username string) {
	oidcResponse := authenticateWithIdp(params)

	unscopedToken = authenticateWithServiceProvider(oidcResponse.BearerToken, params)
	return unscopedToken, oidcResponse.Claims.PreferredUsername
}

func authenticateWithServiceProvider(bearerToken string, authInfo common.AuthInfo) (unscopedToken string) {
	requestPath := endpoints.IdentityProviders(authInfo.IdentityProvider, authInfo.Protocol)

	request, err := http.NewRequest(http.MethodPost, requestPath, strings.NewReader(""))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add(headers.Authorization, bearerToken)

	client := common.GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()

	unscopedToken = common.GetUnscopedTokenFromResponseOrThrow(response)
	return
}
