package saml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"otc-auth/src/common"
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
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/%s/auth", common.AuthUrlIam, params.IdentityProvider, params.Protocol), nil)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Accept", common.SoapContentType)
	request.Header.Add("PAOS", common.SoapHeaderInfo)

	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		common.OutputErrorToConsoleAndExit(err)
	}
	return response
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response, client http.Client) []byte {
	request, err := http.NewRequest("POST", params.IdentityProviderUrl, samlResponse.Body)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-type", common.XmlContentType)
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
	request, err := http.NewRequest("POST", assertionResult.Header.Response.ConsumerUrl, bytes.NewReader(responseBodyBytes))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-type", common.SoapContentType)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 201 {
		common.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()
	return response
}
