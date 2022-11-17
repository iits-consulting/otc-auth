package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	util2 "otc-cli/src/util"
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
		Securitytoken string    `json:"securitytoken"`
	} `json:"credential"`
}

func CreateAccessToken(params AccessTokenCreateParams) {
	println("Creating access token file...")

	resp := performAccessTokenRequest(strconv.Itoa(params.DurationSeconds))
	defer resp.Body.Close()

	// TODO: Do something with the access token!
	respBytes, _ := io.ReadAll(resp.Body)

	var accessTokenCreationResponse AccessTokenCreationResponse
	err := json.Unmarshal(respBytes, &accessTokenCreationResponse)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	accessKeyFileContent := "export OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Access +
		"\nexport AWS_ACCESS_KEY_ID=" + accessTokenCreationResponse.Credential.Access +
		"\nexport OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret +
		"\nexport AWS_SECRET_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret

	util2.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)
	println("Creation finished")
	println("Please source the ak-sk-env.sh in the current directory manually")
}

func performAccessTokenRequest(durationSeconds string) *http.Response {
	unscopedTokenFromFile := util2.ReadOrCreateOTCInfoFromFile().UnscopedToken
	body := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\", \"duration_seconds\": \"%s\"}}}}", unscopedTokenFromFile, durationSeconds)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3.0/OS-CREDENTIAL/securitytokens", IamAuthUrl), strings.NewReader(body))
	if err != nil {
		return nil
	}

	req.Header.Add("Content-Type", util2.JsonContentType)

	client := GetHttpClientWithUnscopedToken()
	resp, err := client.Do(req)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	return resp
}
