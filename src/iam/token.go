package iam

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"io"
	"log"
	"net/http"
	"otc-auth/src/util"
	"strings"
	"time"
)

func getUnscopedSAMLToken(params LoginParams) (unscopedToken string) {
	client := GetHttpClient()

	samlResponse := getSAMLRequestFromServiceProvider(params, client)

	responseBodyBytes := authenticateIdpWithSAML(params, samlResponse, client)

	assertionResult := GetSAMLAssertionResult{}
	err := xml.Unmarshal(responseBodyBytes, &assertionResult)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	validatedSAMLResponse := validateSAMLAuthenticationWithServiceProvider(assertionResult, responseBodyBytes, client)
	unscopedToken = getUnscopedTokenFromResponseOrThrow(validatedSAMLResponse)
	defer validatedSAMLResponse.Body.Close()
	return
}

func getUnscopedOIDCToken(params LoginParams) (unscopedToken string, username string) {
	oidcResponse := AuthenticateWithIdp(params)

	unscopedToken = getUnscopedTokenWithIdpBearerToken(oidcResponse.BearerToken, params)
	return unscopedToken, oidcResponse.Claims.PreferredUsername
}

func getUserToken(params LoginParams) (unscopedToken string) {
	requestBody := getRequestBodyForAuthenticationMethod(params)
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", AuthUrlIam), strings.NewReader(requestBody))
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-Type", util.JsonContentType)

	client := GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()

	unscopedToken = getUnscopedTokenFromResponseOrThrow(response)

	return
}

func GetNewScopedToken(projectName string) {
	projectId := getProjectId(projectName)
	otcInfo := util.ReadOrCreateOTCInfoFromFile()
	err := retry.Do(
		func() error {
			tokenBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", otcInfo.UnscopedToken.Value, projectId)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", AuthUrlIam), strings.NewReader(tokenBody))
			if err != nil {
				util.OutputErrorToConsoleAndExit(err)
			}

			req.Header.Add("Content-Type", util.JsonContentType)

			client := GetHttpClient()
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			scopedToken := resp.Header.Get("X-Subject-Token")

			if scopedToken == "" {
				respBytes, _ := io.ReadAll(resp.Body)
				var formattedJson bytes.Buffer
				err := json.Indent(&formattedJson, respBytes, "", "  ")
				if err != nil {
					util.OutputErrorToConsoleAndExit(err)
				}
				defer resp.Body.Close()
				println("error: an error occurred while polling a scoped token. Will try again")
				return errors.New(fmt.Sprintf("http status code: %s\nresponse body:\n%s", resp.Status, string(formattedJson.Bytes())))
			}

			tokenExpirationDate := time.Now().Add(time.Hour * 23)
			newProjectEntry := util.Project{Name: projectName, ID: projectId, Token: scopedToken, TokenValidTill: tokenExpirationDate.Format(util.TimeFormat)}
			otcInfo.Projects = util.AppendOrReplaceProject(otcInfo.Projects, newProjectEntry)
			util.UpdateOtcInformation(otcInfo)
			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(time.Second*5),
	)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
}

func getUnscopedTokenWithIdpBearerToken(bearerToken string, params LoginParams) (unscopedToken string) {
	requestPath := fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/oidc/auth", AuthUrlIam, params.IdentityProvider)

	request, err := http.NewRequest("POST", requestPath, strings.NewReader(""))
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Authorization", bearerToken)

	client := GetHttpClient()
	response, err := client.Do(request)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()

	unscopedToken = getUnscopedTokenFromResponseOrThrow(response)

	return
}

func getSAMLRequestFromServiceProvider(params LoginParams, client http.Client) *http.Response {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/%s/auth", AuthUrlIam, params.IdentityProvider, params.Protocol), nil)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Accept", SoapContentType)
	request.Header.Add("PAOS", SoapHeaderInfo)

	defer client.CloseIdleConnections()
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		util.OutputErrorToConsoleAndExit(err)
	}
	return response
}

func authenticateIdpWithSAML(params LoginParams, samlResponse *http.Response, client http.Client) []byte {
	request, err := http.NewRequest("POST", params.IdentityProviderUrl, samlResponse.Body)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-type", XmlContentType)
	request.SetBasicAuth(params.Username, params.Password)

	response, err := client.Do(request)
	if err != nil || response.StatusCode != 200 {
		util.OutputErrorToConsoleAndExit(err)
	}

	responseBodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	return responseBodyBytes
}

func validateSAMLAuthenticationWithServiceProvider(assertionResult GetSAMLAssertionResult, responseBodyBytes []byte, client http.Client) *http.Response {
	request, err := http.NewRequest("POST", assertionResult.Header.Response.ConsumerUrl, bytes.NewReader(responseBodyBytes))
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-type", SoapContentType)
	response, err := client.Do(request)
	if err != nil || response.StatusCode != 201 {
		util.OutputErrorToConsoleAndExit(err)
	}
	defer response.Body.Close()
	return response
}

func getUnscopedTokenFromResponseOrThrow(response *http.Response) (unscopedToken string) {
	unscopedToken = response.Header.Get("X-Subject-Token")
	if unscopedToken == "" {
		responseBytes, _ := io.ReadAll(response.Body)
		responseString := string(responseBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			util.OutputErrorMessageToConsoleAndExit("fatal: invalid otp unscopedToken.\n\nPlease try it again with a new otp unscopedToken.")
		} else {
			util.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: response failed with status %s.\n\nBody: %s", response.Status, responseString))
		}
	}
	return unscopedToken
}

func getRequestBodyForAuthenticationMethod(params LoginParams) (requestBody string) {
	if params.Otp != "" && params.UserDomainId != "" {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\", \"totp\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}, "+
			"\"totp\" : {\"user\": {\"id\": \"%s\", \"passcode\": \"%s\"}}}, \"scope\": {\"domain\": {\"name\": \"%s\"}}}}",
			params.Username, params.Password, params.DomainName, params.UserDomainId, params.Otp, params.DomainName)
	} else {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}}, "+
			"\"scope\": {\"domain\": {\"name\": \"%s\"}}}}", params.Username, params.Password, params.DomainName, params.DomainName)
	}
	return requestBody
}
