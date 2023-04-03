package oidc

import (
	"encoding/json"
	"github.com/go-http-utils/headers"
	"net/http"
	"net/url"
	"otc-auth/common"
	"strings"
)

func createServiceAccountAuthenticateRequest(requestUrl string, clientId string, clientSecret string) *http.Request {
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("scope", "openid")
	request := common.GetRequest(http.MethodPost, requestUrl, strings.NewReader(data.Encode()))
	request.SetBasicAuth(clientId, clientSecret)
	request.Header.Add(headers.ContentType, "application/x-www-form-urlencoded")
	return request
}

type ServiceAccountResponse struct {
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	TokenType        string `json:"token_type"`
	IdToken          string `json:"id_token"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshToken     string `json:"refresh_token"`
	Scope            string `json:"scope"`
}

func authenticateServiceAccountWithIdp(params common.AuthInfo) common.OidcCredentialsResponse {
	idpTokenUrl, err := url.JoinPath(params.IdpUrl, "protocol/openid-connect/token")
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	request := createServiceAccountAuthenticateRequest(idpTokenUrl, params.ClientId, params.ClientSecret)
	response := common.HttpClientMakeRequest(request)
	bodyBytes := common.GetBodyBytesFromResponse(response)

	var result ServiceAccountResponse
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return common.OidcCredentialsResponse{
		BearerToken: result.IdToken,
		Claims: struct {
			PreferredUsername string `json:"preferred_username"`
		}(struct{ PreferredUsername string }{PreferredUsername: "ServiceAccount"}),
	}
}
