package saml

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"

	"github.com/go-http-utils/headers"
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

	defer func(Body io.ReadCloser) {
		errClose := Body.Close()
		if errClose != nil {
			common.OutputErrorToConsoleAndExit(errClose)
		}
	}(response.Body)

	return tokenResponse
}

func getServiceProviderInitiatedRequest(params common.AuthInfo) *http.Response {
	request := common.GetRequest(http.MethodGet, endpoints.IdentityProviders(params.IdpName, params.AuthProtocol), nil)
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return common.HTTPClientMakeRequest(request)
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response) []byte {
	request := common.GetRequest(http.MethodPost, params.IdpURL, samlResponse.Body)
	request.Header.Add(headers.ContentType, headervalues.TextXML)
	request.SetBasicAuth(params.Username, params.Password)

	response := common.HTTPClientMakeRequest(request)
	return common.GetBodyBytesFromResponse(response)
}

func validateAuthenticationWithServiceProvider(assertionResult common.SamlAssertionResponse, responseBodyBytes []byte) *http.Response {
	request := common.GetRequest(http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL,
		bytes.NewReader(responseBodyBytes))
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return common.HTTPClientMakeRequest(request)
}
