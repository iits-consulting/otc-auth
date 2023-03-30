package saml

import (
	"bytes"
	"encoding/xml"
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) (tokenResponse common.TokenResponse) {
	spInitiatedRequest := getServiceProviderInitiatedRequest(authInfo)

	bodyBytes := authenticateWithIdp(authInfo, spInitiatedRequest)

	assertionResult := common.SamlAssertionResponse{}
	err := xml.Unmarshal(bodyBytes, &assertionResult)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error deserializing xml.\ntrace: %s")
	}

	response := validateAuthenticationWithServiceProvider(assertionResult, bodyBytes)
	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)
	defer response.Body.Close()
	return
}

func getServiceProviderInitiatedRequest(params common.AuthInfo) *http.Response {
	request := common.GetRequest(http.MethodGet, endpoints.IdentityProviders(params.IdpName, params.AuthProtocol), nil)
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return common.HttpClientMakeRequest(request)
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response) []byte {
	request := common.GetRequest(http.MethodPost, params.IdpUrl, samlResponse.Body)
	request.Header.Add(headers.ContentType, headervalues.TextXml)
	request.SetBasicAuth(params.Username, params.Password)

	response := common.HttpClientMakeRequest(request)
	return common.GetBodyBytesFromResponse(response)
}

func validateAuthenticationWithServiceProvider(assertionResult common.SamlAssertionResponse, responseBodyBytes []byte) *http.Response {
	request := common.GetRequest(http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL, bytes.NewReader(responseBodyBytes))
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return common.HttpClientMakeRequest(request)
}
