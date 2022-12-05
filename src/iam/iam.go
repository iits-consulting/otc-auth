package iam

import (
	"fmt"
	"github.com/avast/retry-go"
	"github.com/go-http-utils/headers"
	"log"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
	"otc-auth/src/common/headervalues"
	"otc-auth/src/common/xheaders"
	"otc-auth/src/config"
	"strings"
	"time"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	requestBody := getRequestBodyForAuthenticationMethod(authInfo)
	request := common.GetRequest(http.MethodPost, endpoints.IamTokens, strings.NewReader(requestBody))
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)

	response := common.HttpClientMakeRequest(request)
	defer response.Body.Close()

	return common.GetCloudCredentialsFromResponseOrThrow(response)
}

func GetScopedTokenFromServiceProvider(projectName string) {
	cloud := config.GetActiveCloudConfig()
	projectId := cloud.Projects.GetProjectByNameOrThrow(projectName).Id

	err := retry.Do(
		func() error {
			requestBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", cloud.Tokens.GetUnscopedToken().Secret, projectId)

			request := common.GetRequest(http.MethodPost, endpoints.IamTokens, strings.NewReader(requestBody))
			request.Header.Add(headers.ContentType, headervalues.ApplicationJson)

			response := common.HttpClientMakeRequest(request)

			scopedToken := response.Header.Get(xheaders.XSubjectToken)

			if scopedToken == "" {
				bodyBytes := common.GetBodyBytesFromResponse(response)
				formattedError := common.ByteSliceToIndentedJsonFormat(bodyBytes)
				defer response.Body.Close()
				println("error: an error occurred while polling a scoped token. Will try again")
				return fmt.Errorf("http status code: %s\nresponse body:\n%s", response.Status, formattedError)
			}

			bodyBytes := common.GetBodyBytesFromResponse(response)
			tokenResponse := common.DeserializeJsonForType[common.TokenResponse](bodyBytes)
			defer response.Body.Close()

			token := config.Token{
				Type:      config.Scoped,
				Secret:    scopedToken,
				IssuedAt:  tokenResponse.Token.IssuedAt,
				ExpiresAt: tokenResponse.Token.ExpiresAt,
			}
			cloud.Tokens.UpdateToken(token)
			config.UpdateCloudConfig(cloud)
			println("scoped token acquired successfully.")

			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(time.Second*5),
	)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
}

func getRequestBodyForAuthenticationMethod(authInfo common.AuthInfo) (requestBody string) {
	if authInfo.Otp != "" && authInfo.UserDomainId != "" {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\", \"totp\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}, "+
			"\"totp\" : {\"user\": {\"id\": \"%s\", \"passcode\": \"%s\"}}}, \"scope\": {\"domain\": {\"name\": \"%s\"}}}}",
			authInfo.Username, authInfo.Password, authInfo.DomainName, authInfo.UserDomainId, authInfo.Otp, authInfo.DomainName)
	} else {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}}, "+
			"\"scope\": {\"domain\": {\"name\": \"%s\"}}}}", authInfo.Username, authInfo.Password, authInfo.DomainName, authInfo.DomainName)
	}
	return requestBody
}
