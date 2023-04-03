package accesstoken

import (
	"fmt"
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"strconv"
	"strings"
)

func CreateAccessToken(durationSeconds int) {
	println("Creating access token file...")

	response := getAccessTokenFromServiceProvider(strconv.Itoa(durationSeconds))
	bodyBytes := common.GetBodyBytesFromResponse(response)

	accessTokenCreationResponse := common.DeserializeJsonForType[TokenCreationResponse](bodyBytes)

	accessKeyFileContent := fmt.Sprintf(
		"export OS_ACCESS_KEY=%s\n"+
			"export AWS_ACCESS_KEY_ID=%s\n"+
			"export OS_SECRET_KEY=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s",
		accessTokenCreationResponse.Credential.Access,
		accessTokenCreationResponse.Credential.Access,
		accessTokenCreationResponse.Credential.Secret,
		accessTokenCreationResponse.Credential.Secret)

	common.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)

	println("Access token file created successfully.")
	println("Please source the ak-sk-env.sh file in the current directory manually")
}

func getAccessTokenFromServiceProvider(durationSeconds string) *http.Response {
	secret := config.GetActiveCloudConfig().UnscopedToken.Secret
	body := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\", \"duration_seconds\": \"%s\"}}}}", secret, durationSeconds)

	request := common.GetRequest(http.MethodPost, endpoints.IamSecurityTokens, strings.NewReader(body))
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	request.Header.Add(xheaders.XAuthToken, secret)

	return common.HttpClientMakeRequest(request)
}
