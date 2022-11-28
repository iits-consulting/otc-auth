package oidc

import (
	"fmt"
	"net/http"
	"otc-auth/src/common"
	"strings"
)

func AuthenticateAndGetUnscopedToken(params common.AuthInfo) (unscopedToken string, username string) {
	oidcResponse := authenticateWithIdp(params)

	unscopedToken = authenticateWithServiceProvider(oidcResponse.BearerToken, params)
	return unscopedToken, oidcResponse.Claims.PreferredUsername
}

func authenticateWithServiceProvider(bearerToken string, params common.AuthInfo) (unscopedToken string) {
	requestPath := fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/oidc/auth", common.AuthUrlIam, params.IdentityProvider)

	request, err := http.NewRequest("POST", requestPath, strings.NewReader(""))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Authorization", bearerToken)

	client := common.GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()

	unscopedToken = common.GetUnscopedTokenFromResponseOrThrow(response)
	return
}
