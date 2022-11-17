package util

import (
	"encoding/json"
	"os"
	"time"
)

const TimeFormat = time.RFC3339

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

func LoginNeeded() bool {
	if !fileExists(otcInfoPath) {
		return true
	}
	otcInformation := ReadOrCreateOTCInfoFromFile()
	if otcInformation.UnscopedToken.ValidTill == "" {
		return true
	}
	validTill, err := time.Parse(TimeFormat, otcInformation.UnscopedToken.ValidTill)
	if err != nil {
		OutputErrorToConsoleAndExit(err)
	}
	if validTill.After(time.Now()) {
		println("Old unscoped token is still valid till: " + otcInformation.UnscopedToken.ValidTill)
		return false
	}
	return false
}

func GetScopedTokenFromOTCInfo(projectName string) string {
	otcInfo := ReadOrCreateOTCInfoFromFile()
	for i := range otcInfo.Projects {
		project := otcInfo.Projects[i]
		if project.Name == projectName {
			tokenValidTill, err := time.Parse(TimeFormat, project.TokenValidTill)
			if err != nil {
				OutputErrorToConsoleAndExit(err)
			}
			if tokenValidTill.After(time.Now()) {
				println("Old scoped token for project " + projectName + " is still valid till: " + project.TokenValidTill)
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
	OutputErrorMessageToConsoleAndExit("Something went wrong. Project with name=" + projectName + " not found inside otc-info file path=" + otcInfoPath)
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
