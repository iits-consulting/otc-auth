package iam

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"github.com/avast/retry-go"
	"io"
	"log"
	"net/http"
	"otc-cli/src/util"
	"strconv"
	"strings"
	"time"
)

func getUnscopedSAMLToken(params LoginParams) (token string) {
	client := &http.Client{}

	samlResponse := getSAMLRequestFromServiceProvider(params, client)

	responseBodyBytes := authenticateIdpWithSAML(params, samlResponse, client)

	assertionResult := GetSAMLAssertionResult{}
	err := xml.Unmarshal(responseBodyBytes, &assertionResult)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	validatedSAMLResponse := validateSAMLAuthenticationWithServiceProvider(assertionResult, responseBodyBytes, client)

	token = getUnscopedTokenFromResponseOrThrow(validatedSAMLResponse)

	return
}

func getUserToken(params LoginParams) (token string) {
	requestBody := getRequestBodyForAuthenticationMethod(params)
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", IamAuthUrl), strings.NewReader(requestBody))
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

	token = getUnscopedTokenFromResponseOrThrow(response)

	return
}

func OrderNewScopedToken(projectName string) {
	projectId := getProjectId(projectName)
	otcInfo := util.ReadOrCreateOTCInfoFromFile()
	err := retry.Do(
		func() error {
			tokenBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", otcInfo.UnscopedToken.Value, projectId)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", IamAuthUrl), strings.NewReader(tokenBody))
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
				defer resp.Body.Close()
				println("OTC API not reachable will try again. Errorcode:")
				return errors.New("Statuscode: " + strconv.Itoa(resp.StatusCode) + ",Body:" + string(respBytes))
			}

			valid23Hours := time.Now().Add(time.Hour)
			newProjectEntry := util.Project{Name: projectName, ID: projectId, Token: scopedToken, TokenValidTill: valid23Hours.Format(util.TimeFormat)}
			otcInfo.Projects = util.AppendOrReplaceProject(otcInfo.Projects, newProjectEntry)
			util.UpdateOtcInformation(otcInfo)
			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(5*time.Second),
	)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
}

func getSAMLRequestFromServiceProvider(params LoginParams, client *http.Client) *http.Response {
	request, err := http.NewRequest("GET", fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/%s/auth", IamAuthUrl, params.IdentityProvider, params.Protocol), nil)
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
	defer response.Body.Close()
	return response
}

func authenticateIdpWithSAML(params LoginParams, samlResponse *http.Response, client *http.Client) []byte {
	serviceProviderRequest, err := http.NewRequest("POST", params.IdentityProviderUrl, samlResponse.Body)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}

	serviceProviderRequest.Header.Add("Content-type", XmlContentType)
	serviceProviderRequest.SetBasicAuth(params.Username, params.Password)

	serviceProviderResponse, err := client.Do(serviceProviderRequest)
	if err != nil || serviceProviderResponse.StatusCode != 200 {
		util.OutputErrorToConsoleAndExit(err)
	}
	defer serviceProviderResponse.Body.Close()

	responseBodyBytes, err := io.ReadAll(serviceProviderResponse.Body)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	return responseBodyBytes
}

func validateSAMLAuthenticationWithServiceProvider(assertionResult GetSAMLAssertionResult, responseBodyBytes []byte, client *http.Client) *http.Response {
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

func getUnscopedTokenFromResponseOrThrow(response *http.Response) (token string) {
	token = response.Header.Get("X-Subject-Token")
	if token == "" {
		responseBytes, _ := io.ReadAll(response.Body)
		defer response.Body.Close()
		responseString := string(responseBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			util.OutputErrorMessageToConsoleAndExit("fatal: invalid otp token.\n\nPlease try it again with a new otp token.")
		} else {
			util.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: response failed with status %s.\n\nBody: %s", response.Status, responseString))
		}
	}
	return token
}

func getRequestBodyForAuthenticationMethod(params LoginParams) (requestBody string) {
	if params.Otp != "" && params.UserId != "" {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\", \"totp\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}, "+
			"\"totp\" : {\"user\": {\"id\": \"%s\", \"passcode\": \"%s\"}}}, \"scope\": {\"domain\": {\"name\": \"%s\"}}}}",
			params.Username, params.Password, params.DomainName, params.UserId, params.Otp, params.DomainName)
	} else {
		requestBody = fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"password\"], "+
			"\"password\": {\"user\": {\"name\": \"%s\", \"password\": \"%s\", \"domain\": {\"name\": \"%s\"}}}}, "+
			"\"scope\": {\"domain\": {\"name\": \"%s\"}}}}", params.Username, params.Password, params.DomainName, params.DomainName)
	}
	return requestBody
}
