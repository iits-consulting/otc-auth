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

type Authenticator struct {
	client common.HTTPClient
	parser CredentialParser
}

func newAuthenticator(client common.HTTPClient, parser CredentialParser) *Authenticator {
	return &Authenticator{
		client: client,
		parser: parser,
	}
}

func AuthenticateAndGetUnscopedToken(ctx context.Context,
	authInfo common.AuthInfo,
) (*common.TokenResponse, error) {
	client := common.NewHTTPClient(authInfo.SkipTLS)
	parser := NewDefaultCredentialParser()
	service := newAuthenticator(client, parser)
	return service.Authenticate(ctx, authInfo)
}

func (a *Authenticator) Authenticate(ctx context.Context, authInfo common.AuthInfo) (*common.TokenResponse, error) {
	spInitiatedRequest, err := a.getServiceProviderInitiatedRequest(ctx, authInfo)
	if err != nil {
		return nil, fmt.Errorf("error getting sp request\ntrace: %w", err)
	}
	defer spInitiatedRequest.Body.Close()

	bodyBytes, err := a.authenticateWithIdp(ctx, authInfo, spInitiatedRequest)
	if err != nil {
		return nil, fmt.Errorf("couldn't auth with idp: %w", err)
	}

	assertionResult := common.SamlAssertionResponse{}

	err = xml.Unmarshal(bodyBytes, &assertionResult)
	if err != nil {
		return nil, fmt.Errorf("fatal: error deserializing xml.\ntrace: %w", err)
	}

	response, err := a.validateAuthenticationWithServiceProvider(ctx, assertionResult, bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("couldn't validate auth with service provider: %w", err)
	}
	defer response.Body.Close()

	tokenResponse, err := a.parser.Parse(response)
	if err != nil {
		return nil, fmt.Errorf("couldn't get cloud creds from response: %w", err)
	}

	return tokenResponse, nil
}

func (a *Authenticator) getServiceProviderInitiatedRequest(ctx context.Context,
	params common.AuthInfo,
) (*http.Response, error) {
	request, err := common.NewRequest(ctx, http.MethodGet,
		endpoints.IdentityProviders(params.IdpName, string(params.AuthProtocol), params.Region), nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add(headers.Accept, headervalues.ApplicationPaos)
	request.Header.Add(header.Paos, headervalues.Paos)

	return a.client.MakeRequest(request)
}

func (a *Authenticator) authenticateWithIdp(ctx context.Context, params common.AuthInfo,
	samlResponse *http.Response,
) ([]byte, error) {
	request, err := common.NewRequest(ctx, http.MethodPost, params.IdpURL, samlResponse.Body)
	if err != nil {
		return nil, err
	}
	defer request.Body.Close()

	request.Header.Add(headers.ContentType, headervalues.TextXML)
	request.SetBasicAuth(params.Username, params.Password)

	response, err := a.client.MakeRequest(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	return common.GetBodyBytesFromResponse(response)
}

func (a *Authenticator) validateAuthenticationWithServiceProvider(ctx context.Context,
	assertionResult common.SamlAssertionResponse,
	responseBodyBytes []byte,
) (*http.Response, error) {
	request, err := common.NewRequest(ctx, http.MethodPost, assertionResult.Header.Response.AssertionConsumerServiceURL,
		bytes.NewReader(responseBodyBytes))
	if err != nil {
		return nil, err
	}
	request.Header.Add(headers.ContentType, headervalues.ApplicationPaos)

	return a.client.MakeRequest(request)
}
