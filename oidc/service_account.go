package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"otc-auth/common"

	"github.com/go-http-utils/headers"
)

func createServiceAccountAuthenticateRequest(ctx context.Context, requestURL string,
	clientID string, clientSecret string,
) (*http.Request, error) {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "openid")
	request, err := common.NewRequest(ctx, http.MethodPost, requestURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	request.SetBasicAuth(clientID, clientSecret)
	request.Header.Add(headers.ContentType, "application/x-www-form-urlencoded")
	return request, nil
}

type ServiceAccountResponse struct {
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	IDToken          string `json:"id_token"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	Scope            string `json:"scope"`
}

func authenticateServiceAccountWithIdp(ctx context.Context, params common.AuthInfo, client common.HTTPClient,
) (*common.OidcCredentialsResponse, error) {
	idpTokenURL, err := url.JoinPath(params.IdpURL, "protocol/openid-connect/token")
	if err != nil {
		return nil, err
	}
	request, err := createServiceAccountAuthenticateRequest(ctx, idpTokenURL, params.ClientID, params.ClientSecret)
	if err != nil {
		return nil, err
	}
	response, err := client.MakeRequest(request)
	if err != nil {
		return nil, err
	}
	bodyBytes, err := common.GetBodyBytesFromResponse(response)
	if err != nil {
		return nil, err
	}

	var result ServiceAccountResponse
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return nil, err
	}

	serviceAccountCreds := common.OidcCredentialsResponse{}
	serviceAccountCreds.BearerToken = result.IDToken
	serviceAccountCreds.Claims.PreferredUsername = "ServiceAccount"

	err = response.Body.Close()
	if err != nil {
		return nil, err
	}
	return &serviceAccountCreds, nil
}
