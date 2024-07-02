package saml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"

	"github.com/go-http-utils/headers"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo, skipTLS bool) (tokenResponse common.TokenResponse) {
	spInitiatedRequest := getServiceProviderInitiatedRequest(authInfo, skipTLS) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon

	bodyBytes := authenticateWithIdp(authInfo, spInitiatedRequest, skipTLS)

	assertionResult := common.SamlAssertionResponse{}

	err := xml.Unmarshal(bodyBytes, &assertionResult)
	if err != nil {
		common.ThrowError(fmt.Errorf("fatal: error deserializing xml.\ntrace: %w", err))
	}

	response := validateAuthenticationWithServiceProvider(assertionResult, bodyBytes, skipTLS) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon
	tokenResponse = common.GetCloudCredentialsFromResponseOrThrow(response)

	defer func(Body io.ReadCloser) {
		errClose := Body.Close()
		if errClose != nil {
			common.ThrowError(errClose)
		}
	}(response.Body)

	return tokenResponse
}

func getServiceProviderInitiatedRequest(params common.AuthInfo, skipTLS bool) *http.Response {
	request := common.GetRequest(http.MethodGet,
		endpoints.IdentityProviders(params.IdpName, params.AuthProtocol, params.Region), nil)
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return common.HTTPClientMakeRequest(request, skipTLS)
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response, skipTLS bool) []byte {
	request := common.GetRequest(http.MethodPost, params.IdpURL, samlResponse.Body)
	request.Header.Add(headers.ContentType, headervalues.TextXML)
	request.SetBasicAuth(params.Username, params.Password)

	response := common.HTTPClientMakeRequest(request, skipTLS) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon
	return common.GetBodyBytesFromResponse(response)
}

//nolint:lll // This function will be removed soon
func validateAuthenticationWithServiceProvider(assertionResult common.SamlAssertionResponse, responseBodyBytes []byte, skipTLS bool) *http.Response {
	request := common.GetRequest(http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL,
		bytes.NewReader(responseBodyBytes))
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return common.HTTPClientMakeRequest(request, skipTLS)
}
