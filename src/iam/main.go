package iam

import (
	"fmt"
	"github.com/avast/retry-go"
	"io"
	"log"
	"net/http"
	"otc-auth/src/common"
	"strings"
	"time"
)

func AuthenticateAndGetUnscopedToken(params common.AuthInfo) (unscopedToken string) {
	requestBody := getRequestBodyForAuthenticationMethod(params)
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", common.AuthUrlIam), strings.NewReader(requestBody))
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-Type", common.JsonContentType)

	client := common.GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		respBytes, _ := io.ReadAll(response.Body)
		formattedError := common.ErrorMessageToIndentedJsonFormat(respBytes)
		common.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: authentication failed with status %s. response:\n\n%s", response.Status, formattedError))
	}
	defer response.Body.Close()

	unscopedToken = common.GetUnscopedTokenFromResponseOrThrow(response)

	return
}

func GetScopedToken(projectName string) {
	projectId := getProjectId(projectName)
	otcInfo := common.ReadOrCreateOTCAuthCredentialsFile()
	err := retry.Do(
		func() error {
			tokenBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", otcInfo.UnscopedToken.Value, projectId)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", common.AuthUrlIam), strings.NewReader(tokenBody))
			if err != nil {
				common.OutputErrorToConsoleAndExit(err)
			}

			req.Header.Add("Content-Type", common.JsonContentType)

			client := common.GetHttpClient()
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			scopedToken := resp.Header.Get("X-Subject-Token")

			if scopedToken == "" {
				respBytes, _ := io.ReadAll(resp.Body)
				formattedError := common.ErrorMessageToIndentedJsonFormat(respBytes)
				defer resp.Body.Close()
				println("error: an error occurred while polling a scoped token. Will try again")
				return fmt.Errorf("http status code: %s\nresponse body:\n%s", resp.Status, formattedError)
			}

			tokenExpirationDate := time.Now().Add(time.Hour * 23)
			newProjectEntry := common.Project{Name: projectName, ID: projectId, Token: scopedToken, TokenValidTill: tokenExpirationDate.Format(common.TimeFormat)}
			otcInfo.Projects = common.AppendOrReplaceProject(otcInfo.Projects, newProjectEntry)
			common.UpdateOtcInformation(otcInfo)
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
