package oidc

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"

	"otc-auth/common"

	"github.com/go-http-utils/headers"
)

func createServiceAccountAuthenticateRequest(requestURL string, clientID string, clientSecret string) *http.Request {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "openid")
	request := common.GetRequest(http.MethodPost, requestURL, strings.NewReader(data.Encode()))
	request.SetBasicAuth(clientID, clientSecret)
	request.Header.Add(headers.ContentType, "application/x-www-form-urlencoded")
	return request
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

func authenticateServiceAccountWithIdp(params common.AuthInfo, skipTLS bool) common.OidcCredentialsResponse {
	idpTokenURL, err := url.JoinPath(params.IdpURL, "protocol/openid-connect/token")
	if err != nil {
		log.Fatal(err)
	}
	request := createServiceAccountAuthenticateRequest(idpTokenURL, params.ClientID, params.ClientSecret)
	response := common.HTTPClientMakeRequest(request, skipTLS) //nolint:bodyclose,lll // Works fine for now, this method will be replaced soon
	bodyBytes := common.GetBodyBytesFromResponse(response)

	var result ServiceAccountResponse
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		log.Fatal(err)
	}

	serviceAccountCreds := common.OidcCredentialsResponse{}
	serviceAccountCreds.BearerToken = result.IDToken
	serviceAccountCreds.Claims.PreferredUsername = "ServiceAccount"

	return serviceAccountCreds
}
