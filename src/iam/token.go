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
	util2 "otc-cli/src/util"
	"strconv"
	"strings"
	"time"
)

func getUnscopedSAMLToken(params LoginParams) (token string) {
	idpReq, err := http.NewRequest("GET", fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers/%s/protocols/%s/auth", IamAuthUrl, params.IdentityProvider, params.Protocol), nil)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	idpReq.Header.Add("Accept", SoapContentType)
	idpReq.Header.Add("PAOS", SoapHeaderInfo)

	client := &http.Client{}
	defer client.CloseIdleConnections()
	idpResp, err := client.Do(idpReq)
	if err != nil || idpResp.StatusCode != 200 {
		util2.OutputErrorToConsoleAndExit(err)
	}
	defer idpResp.Body.Close()

	serviceProviderRequest, err := http.NewRequest("POST", params.IdentityProviderUrl, idpResp.Body)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	serviceProviderRequest.Header.Add("Content-type", XmlContentType)
	serviceProviderRequest.SetBasicAuth(params.Username, params.Password)

	serviceProviderResponse, err := client.Do(serviceProviderRequest)
	if err != nil || serviceProviderResponse.StatusCode != 200 {
		util2.OutputErrorToConsoleAndExit(err)
	}
	defer serviceProviderResponse.Body.Close()

	spRespBodyBytes, err := io.ReadAll(serviceProviderResponse.Body)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	assertionResult := GetSAMLAssertionResult{}
	err = xml.Unmarshal(spRespBodyBytes, &assertionResult)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	samlReq, err := http.NewRequest("POST", assertionResult.Header.Response.ConsumerUrl, bytes.NewReader(spRespBodyBytes))
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	samlReq.Header.Add("Content-type", SoapContentType)
	samlResp, err := client.Do(samlReq)
	if err != nil || samlResp.StatusCode != 201 {
		util2.OutputErrorToConsoleAndExit(err)
	}
	defer samlResp.Body.Close()

	token = samlResp.Header.Get("X-Subject-Token")
	if token == "" {
		respBytes, _ := io.ReadAll(samlResp.Body)
		defer samlResp.Body.Close()
		util2.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: response failed with status %s.\n\nBody: %s", samlResp.Status, string(respBytes)))
	}

	return
}

func getUserToken(params LoginParams) (token string) {
	var requestBody string
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
	request, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", IamAuthUrl), strings.NewReader(requestBody))
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}

	request.Header.Add("Content-Type", util2.JsonContentType)

	client := GetHttpClient()
	resp, err := client.Do(request)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}
	defer resp.Body.Close()

	token = resp.Header.Get("X-Subject-Token")
	if token == "" {
		respBytes, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		responseString := string(respBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			util2.OutputErrorMessageToConsoleAndExit("fatal: invalid otp token.\n\nPlease try it again with a new otp token.")
		} else {
			util2.OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: response failed with status %s.\n\nBody: %s", resp.Status, responseString))
		}

	}

	return
}

func OrderNewScopedToken(projectName string) {
	projectId := getProjectId(projectName)
	otcInfo := util2.ReadOrCreateOTCInfoFromFile()
	err := retry.Do(
		func() error {
			tokenBody := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\"}}, \"scope\": {\"project\": {\"id\": \"%s\"}}}}", otcInfo.UnscopedToken.Value, projectId)

			req, err := http.NewRequest("POST", fmt.Sprintf("%s/v3/auth/tokens", IamAuthUrl), strings.NewReader(tokenBody))
			if err != nil {
				util2.OutputErrorToConsoleAndExit(err)
			}

			req.Header.Add("Content-Type", util2.JsonContentType)

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
			newProjectEntry := util2.Project{Name: projectName, ID: projectId, Token: scopedToken, TokenValidTill: valid23Hours.Format(util2.TimeFormat)}
			otcInfo.Projects = util2.AppendOrReplaceProject(otcInfo.Projects, newProjectEntry)
			util2.UpdateOtcInformation(otcInfo)
			return nil
		}, retry.OnRetry(func(n uint, err error) {
			log.Printf("#%d: %s\n", n, err)
		}),
		retry.DelayType(retry.FixedDelay),
		retry.Delay(5*time.Second),
	)
	if err != nil {
		util2.OutputErrorToConsoleAndExit(err)
	}
}
