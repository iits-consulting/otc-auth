package saml

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	header "otc-auth/common/xheaders"

	"github.com/go-http-utils/headers"
)

func AuthenticateAndGetUnscopedToken(ctx context.Context, authInfo common.AuthInfo) (*common.TokenResponse, error) {
	httpClient := common.NewHTTPClient(authInfo.SkipTLS)
	spInitiatedRequest, err := getServiceProviderInitiatedRequest(ctx, authInfo, httpClient)
	if err != nil {
		return nil, fmt.Errorf("error getting sp request\ntrace: %w", err)
	}
	defer spInitiatedRequest.Body.Close()

	bodyBytes, err := authenticateWithIdp(ctx, authInfo, spInitiatedRequest, httpClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't auth with idp: %w", err)
	}

	assertionResult := common.SamlAssertionResponse{}

	err = xml.Unmarshal(bodyBytes, &assertionResult)
	if err != nil {
		return nil, fmt.Errorf("fatal: error deserializing xml.\ntrace: %w", err)
	}

	response, err := validateAuthenticationWithServiceProvider(ctx, assertionResult, bodyBytes, httpClient)
	if err != nil {
		return nil, fmt.Errorf("couldn't validate auth with service provider: %w", err)
	}
	defer response.Body.Close()

	tokenResponse, err := common.GetCloudCredentialsFromResponse(response)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cloud creds from response: %w", err)
	}

	return tokenResponse, nil
}

func getServiceProviderInitiatedRequest(ctx context.Context,
	params common.AuthInfo, client common.HTTPClient,
) (*http.Response, error) {
	request, err := common.NewRequest(ctx, http.MethodGet,
		endpoints.IdentityProviders(params.IdpName, string(params.AuthProtocol), params.Region), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return client.MakeRequest(request)
}

func authenticateWithIdp(ctx context.Context, params common.AuthInfo,
	samlResponse *http.Response, client common.HTTPClient,
) ([]byte, error) {
	request, err := common.NewRequest(ctx, http.MethodPost, params.IdpURL, samlResponse.Body)
	if err != nil {
		return nil, err
	}
	defer request.Body.Close()

	request.Header.Add(headers.ContentType, headervalues.TextXML)
	request.SetBasicAuth(params.Username, params.Password)

	response, err := client.MakeRequest(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return common.GetBodyBytesFromResponse(response)
}

func validateAuthenticationWithServiceProvider(ctx context.Context, assertionResult common.SamlAssertionResponse,
	responseBodyBytes []byte, client common.HTTPClient,
) (*http.Response, error) {
	request, err := common.NewRequest(ctx, http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL,
		bytes.NewReader(responseBodyBytes))
	if err != nil {
		return nil, err
	}
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return client.MakeRequest(request)
}
