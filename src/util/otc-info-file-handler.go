package util

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const (
	TimeFormat      = time.RFC3339
	PrintTimeFormat = time.RFC1123
)

var otcInfoPath = GetHomeDir() + "/.otc-info"

type OtcInfo struct {
	UnscopedToken UnscopedToken `json:"unscopedToken"`
	Username      string        `json:"username"`
	Projects      []Project     `json:"projects"`
}

type Project struct {
	Name           string `json:"name"`
	ID             string `json:"id"`
	Token          string `json:"token"`
	TokenValidTill string `json:"tokenValidTill"`
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

type UnscopedToken struct {
	Value     string `json:"value"`
	ValidTill string `json:"validTill"`
}

func LoginNeeded(overwrite bool) bool {
	if !fileExists(otcInfoPath) {
		return true
	}
	otcInformation := ReadOrCreateOTCInfoFromFile()
	if otcInformation.UnscopedToken.ValidTill == "" {
		return true
	}
	tokenExpirationDate, err := time.Parse(TimeFormat, otcInformation.UnscopedToken.ValidTill)
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
	otcInfo := ReadOrCreateOTCInfoFromFile()
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

func ReadOrCreateOTCInfoFromFile() OtcInfo {
	var otcInfo OtcInfo
	otcLoginInfoData, err := ReadFileContent(otcInfoPath)

	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	if otcLoginInfoData == "" {
		otcLoginInfoData = readOrCreateOTCInfoFile(otcInfo)
	}
	err = json.Unmarshal([]byte(otcLoginInfoData), &otcInfo)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}

	return otcInfo
}

func UpdateOtcInformation(otcInformation OtcInfo) {
	otcInfoData, err := json.Marshal(otcInformation)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	WriteStringToFile(otcInfoPath, string(otcInfoData))
}

func FindProjectID(projectName string) string {
	otcInfo := ReadOrCreateOTCInfoFromFile()
	for i := range otcInfo.Projects {
		project := otcInfo.Projects[i]
		if project.Name == projectName {
			return project.ID
		}
	}
	OutputErrorMessageToConsoleAndExit(fmt.Sprintf("Something went wrong. Project \"%s\" not found in otc-info file located at %s", projectName, otcInfoPath))
	return ""
}

func readOrCreateOTCInfoFile(otcInfo OtcInfo) string {
	result, err := json.Marshal(otcInfo)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	otcInfoData := string(result)
	WriteStringToFile(otcInfoPath, otcInfoData)
	return otcInfoData
}
