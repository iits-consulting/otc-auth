package saml

import (
	"bytes"
	"encoding/xml"
	"github.com/go-http-utils/headers"
	"io"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
	"otc-auth/src/common/headervalues"
	header "otc-auth/src/common/xheaders"
)

func AuthenticateAndGetUnscopedToken(params common.AuthInfo) (unscopedToken string) {
	client := common.GetHttpClient()

	spInitiatedRequest := getServiceProviderInitiatedRequest(params, client)

	responseBodyBytes := authenticateWithIdp(params, spInitiatedRequest, client)

	assertionResult := common.GetSAMLAssertionResult{}
	err := xml.Unmarshal(responseBodyBytes, &assertionResult)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	validatedResponse := validateAuthenticationWithServiceProvider(assertionResult, responseBodyBytes, client)
	unscopedToken = common.GetUnscopedTokenFromResponseOrThrow(validatedResponse)
	defer validatedResponse.Body.Close()
	return
}

func getServiceProviderInitiatedRequest(params common.AuthInfo, client http.Client) *http.Response {
	request, err := http.NewRequest(http.MethodGet, endpoints.IdentityProviders(params.IdentityProvider, params.Protocol), nil)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		common.OutputErrorToConsoleAndExit(err)
	}
	return response
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response, client http.Client) []byte {
	request, err := http.NewRequest(http.MethodPost, params.IdentityProviderUrl, samlResponse.Body)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add(headers.ContentType, headervalues.TextXml)
	request.SetBasicAuth(params.Username, params.Password)

	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		common.OutputErrorToConsoleAndExit(err)
	}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	return responseBodyBytes
}

func validateAuthenticationWithServiceProvider(assertionResult common.GetSAMLAssertionResult, responseBodyBytes []byte, client http.Client) *http.Response {
	request, err := http.NewRequest(http.MethodPost, assertionResult.Header.Response.ConsumerUrl, bytes.NewReader(responseBodyBytes))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 201 {
		common.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()
	return response
}
