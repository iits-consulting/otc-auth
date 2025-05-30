package saml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"net/http"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"

	"github.com/go-http-utils/headers"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo, skipTLS bool) common.TokenResponse {
	spInitiatedRequest, err := getServiceProviderInitiatedRequest(authInfo, skipTLS, common.HTTPClientImpl{}) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon
	if err != nil {
		common.ThrowError(fmt.Errorf("fatal: error getting sp request\ntrace: %w", err))
	}

	bodyBytes, err := authenticateWithIdp(authInfo, spInitiatedRequest, skipTLS, common.HTTPClientImpl{})
	if err != nil {
		common.ThrowError(err)
	}

	assertionResult := common.SamlAssertionResponse{}

	err = xml.Unmarshal(bodyBytes, &assertionResult)
	if err != nil {
		common.ThrowError(fmt.Errorf("fatal: error deserializing xml.\ntrace: %w", err))
	}

	response, err := validateAuthenticationWithServiceProvider(assertionResult, bodyBytes, skipTLS, common.HTTPClientImpl{}) //nolint:bodyclose,lll // The body IS closed later on after being read in GetCloudCredentialsFromResponse. This isn't super neat and might be worth refactoring
	if err != nil {
		common.ThrowError(err)
	}
	tokenResponse, err := common.GetCloudCredentialsFromResponse(response)
	if err != nil {
		common.ThrowError(err)
	}

	return *tokenResponse
}

func getServiceProviderInitiatedRequest(params common.AuthInfo, skipTLS bool, client common.HTTPClient) (*http.Response, error) {
	request := common.GetRequest(http.MethodGet,
		endpoints.IdentityProviders(params.IdpName, params.AuthProtocol, params.Region), nil)
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return client.MakeRequest(request, skipTLS)
}

func authenticateWithIdp(params common.AuthInfo, samlResponse *http.Response, skipTLS bool, client common.HTTPClient) ([]byte, error) {
	request := common.GetRequest(http.MethodPost, params.IdpURL, samlResponse.Body)
	request.Header.Add(headers.ContentType, headervalues.TextXML)
	request.SetBasicAuth(params.Username, params.Password)

	response, err := client.MakeRequest(request, skipTLS) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon
	if err != nil {
		return nil, err
	}
	return common.GetBodyBytesFromResponse(response)
}

//nolint:lll // This function will be removed soon
func validateAuthenticationWithServiceProvider(assertionResult common.SamlAssertionResponse, responseBodyBytes []byte, skipTLS bool, client common.HTTPClient) (*http.Response, error) {
	request := common.GetRequest(http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL,
		bytes.NewReader(responseBodyBytes))
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return client.MakeRequest(request, skipTLS)
}
