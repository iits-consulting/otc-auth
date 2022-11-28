package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	TimeFormat      = time.RFC3339
	PrintTimeFormat = time.RFC1123
)

var otcAuthFilePath = GetHomeDir() + "/.otc-auth"

func GetUnscopedTokenFromResponseOrThrow(response *http.Response) (unscopedToken string) {
	unscopedToken = response.Header.Get("X-Subject-Token")
	if unscopedToken == "" {
		responseBytes, _ := io.ReadAll(response.Body)
		responseString := string(responseBytes)
		if strings.Contains(responseString, "mfa totp code verify fail") {
			OutputErrorMessageToConsoleAndExit("fatal: invalid otp unscopedToken.\n\nPlease try it again with a new otp unscopedToken.")
		} else {
			formattedError := ErrorMessageToIndentedJsonFormat(responseBytes)
			OutputErrorMessageToConsoleAndExit(fmt.Sprintf("fatal: response failed with status %s. Body:\n%s", response.Status, formattedError))
		}
	}
	return unscopedToken
}

func AppendOrReplaceProject(projects []Project, newProject Project) []Project {
	var newProjects []Project
	newProjects = append(newProjects, newProject)
	for _, project := range projects {
		if project.ID != newProject.ID {
			newProjects = append(newProjects, project)
		}
	}
	return newProjects
}

func IsAuthenticationValid(overwrite bool) bool {
	if !fileExists(otcAuthFilePath) {
		return true
	}
	otcInfoFile := ReadOrCreateOTCAuthCredentialsFile()
	if otcInfoFile.UnscopedToken.ValidTill == "" {
		return true
	}
	tokenExpirationDate, err := time.Parse(TimeFormat, otcInfoFile.UnscopedToken.ValidTill)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	if tokenExpirationDate.After(time.Now()) {
		println(fmt.Sprintf("Unscoped token is still valid until: %s", tokenExpirationDate.Format(PrintTimeFormat)))
		if overwrite {
			println("Overwriting unscoped token...")
			return true
		}
		return false
	}
	return true
}

func GetScopedTokenFromOTCInfo(projectName string) string {
	otcInfo := ReadOrCreateOTCAuthCredentialsFile()
	for i := range otcInfo.Projects {
		project := otcInfo.Projects[i]
		if project.Name == projectName {
			tokenExpirationDate, err := time.Parse(TimeFormat, project.TokenValidTill)
			if err != nil {
				OutputErrorToConsoleAndExit(err)
			}
			if tokenExpirationDate.After(time.Now()) {
				println(fmt.Sprintf("Scoped token for project %s is still valid till: %s.", projectName, tokenExpirationDate.Format(PrintTimeFormat)))
				return project.Token
			}
			break
		}
	}
	return ""
}

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func ReadOrCreateOTCAuthCredentialsFile() OtcAuthCredentials {
	var otcInfo OtcAuthCredentials
	content, err := ReadFileContent(otcAuthFilePath)

	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	if content == "" {
		content = readOrCreateOTCInfoFile(otcInfo)
	}
	err = json.Unmarshal([]byte(content), &otcInfo)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}

	return otcInfo
}

func UpdateOtcInformation(otcInformation OtcAuthCredentials) {
	otcInfoData, err := json.Marshal(otcInformation)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	WriteStringToFile(otcAuthFilePath, string(otcInfoData))
}

func FindProjectID(projectName string) string {
	otcInfo := ReadOrCreateOTCAuthCredentialsFile()
	for i := range otcInfo.Projects {
		project := otcInfo.Projects[i]
		if project.Name == projectName {
			return project.ID
		}
	}
	OutputErrorMessageToConsoleAndExit(fmt.Sprintf("Something went wrong. Project \"%s\" not found in otc-info file located at %s", projectName, otcAuthFilePath))
	return ""
}

func readOrCreateOTCInfoFile(otcInfo OtcAuthCredentials) string {
	result, err := json.Marshal(otcInfo)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	otcInfoData := string(result)
	WriteStringToFile(otcAuthFilePath, otcInfoData)
	return otcInfoData
}
