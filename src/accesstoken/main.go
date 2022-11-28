package accesstoken

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"otc-auth/src/common"
	"strconv"
	"strings"
)

func CreateAccessToken(params TokenCreateArgs) {
	println("Creating access token file...\n")

	resp := getAccessTokenFromServiceProvider(strconv.Itoa(params.DurationSeconds))
	defer resp.Body.Close()

	// TODO: Do something with the access token!
	respBytes, _ := io.ReadAll(resp.Body)

	var accessTokenCreationResponse TokenCreationResponse
	err := json.Unmarshal(respBytes, &accessTokenCreationResponse)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	accessKeyFileContent := "export OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Access +
		"\nexport AWS_ACCESS_KEY_ID=" + accessTokenCreationResponse.Credential.Access +
		"\nexport OS_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret +
		"\nexport AWS_SECRET_ACCESS_KEY=" + accessTokenCreationResponse.Credential.Secret

	common.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)
	println("Creation finished.\n")
	println("Please source the ak-sk-env.sh in the current directory manually")
}

func getAccessTokenFromServiceProvider(durationSeconds string) *http.Response {
	unscopedTokenFromFile := common.ReadOrCreateOTCAuthCredentialsFile().UnscopedToken
	body := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\", \"duration_seconds\": \"%s\"}}}}", unscopedTokenFromFile, durationSeconds)

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3.0/OS-CREDENTIAL/securitytokens", common.AuthUrlIam), strings.NewReader(body))
	if err != nil {
		return nil
	}

	req.Header.Add("Content-Type", common.JsonContentType)

	client := common.GetHttpClientWithUnscopedToken()
	resp, err := client.Do(req)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return resp
}
