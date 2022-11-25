package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"otc-auth/src/util"
	"strconv"
	"strings"
	"time"
)

type AccessTokenCreateParams struct {
	DurationSeconds int
}

type AccessTokenCreationResponse struct {
	Credential struct {
		Access        string    `json:"access"`
		ExpiresAt     time.Time `json:"expires_at"`
		Secret        string    `json:"secret"`
		SecurityToken string    `json:"securitytoken"`
	} `json:"credential"`
}

func CreateAccessToken(params AccessTokenCreateParams) {
	println("Creating access token file...\n")

	resp := performAccessTokenRequest(strconv.Itoa(params.DurationSeconds))
	defer resp.Body.Close()

	// TODO: Do something with the access token!
	respBytes, _ := io.ReadAll(resp.Body)

	var accessTokenCreationResponse AccessTokenCreationResponse
	err := json.Unmarshal(respBytes, &accessTokenCreationResponse)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	accessKeyFileContent := "export OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Access +
		"\nexport AWS_ACCESS_KEY_ID=" + accessTokenCreationResponse.Credential.Access +
		"\nexport OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret +
		"\nexport AWS_SECRET_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret

	util.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)
	println("Creation finished.\n")
	println("Please source the ak-sk-env.sh in the current directory manually")
}

func performAccessTokenRequest(durationSeconds string) *http.Response {
	unscopedTokenFromFile := util.ReadOrCreateOTCInfoFromFile().UnscopedToken
	body := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\", \"duration_seconds\": \"%s\"}}}}", unscopedTokenFromFile, durationSeconds)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3.0/OS-CREDENTIAL/securitytokens", AuthUrlIam), strings.NewReader(body))
	if err != nil {
		return nil
	}

	req.Header.Add("Content-Type", util.JsonContentType)

	client := GetHttpClientWithUnscopedToken()
	resp, err := client.Do(req)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	return resp
}
