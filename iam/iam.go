package iam

import (
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/go-http-utils/headers"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
	"log"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"strings"
	"time"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) (tokenResponse common.TokenResponse) {
	authOpts := golangsdk.AuthOptions{
		DomainName:       authInfo.DomainName,
		Username:         authInfo.Username,
		Password:         authInfo.Password,
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3"}

	if authInfo.Otp != "" && authInfo.UserDomainId != "" {
		// TODO
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	token, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	user, err := tokens.Create(client, &authOpts).ExtractUser()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	tokenResponse.Token.Secret = token.ID
	tokenResponse.Token.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
	tokenResponse.Token.User.Domain.Id = user.Domain.ID
	tokenResponse.Token.User.Domain.Name = user.Domain.Name
	tokenResponse.Token.User.Name = user.Name
	// TODO time issued?? Is this used?
	return tokenResponse
}

func GetScopedToken(projectName string) config.Token {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)

	if project.ScopedToken.IsTokenValid() {
		token := project.ScopedToken

		tokenExpirationDate := common.ParseTimeOrThrow(token.ExpiresAt)
		if tokenExpirationDate.After(time.Now()) {
			println(fmt.Sprintf("info: scoped token is valid until %s", tokenExpirationDate.Format(common.PrintTimeFormat)))
			return token
		}
	}

	println("attempting to request a scoped token.")
	getScopedTokenFromServiceProvider(projectName)
	project = config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	return project.ScopedToken
}

func getScopedTokenFromServiceProvider(projectName string) {
	cloud := config.GetActiveCloudConfig()
	projectId := cloud.Projects.GetProjectByNameOrThrow(projectName).Id

	err := retry.Do(
		func() error {
			requestBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", cloud.UnscopedToken.Secret, projectId)

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
				Secret:    scopedToken,
				IssuedAt:  tokenResponse.Token.IssuedAt,
				ExpiresAt: tokenResponse.Token.ExpiresAt,
			}
			index := cloud.Projects.FindProjectIndexByName(projectName)
			if index == nil {
				errorMessage := fmt.Sprintf("fatal: project with name %s not found.\n\nUse the cce list-projects command to get a list of projects.", projectName)
				common.OutputErrorToConsoleAndExit(errors.New(errorMessage))
			}
			cloud.Projects[*index].ScopedToken = token
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
