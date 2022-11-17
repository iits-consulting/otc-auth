package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"otc-cli/util"
)

func getProjects() (resp *http.Response, err error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v3/auth/projects", IamAuthUrl), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", util.JsonContentType)

	client := GetHttpClientWithUnscopedToken()
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}

func getProjectId(projectName string) string {
	projectId := ""
	resp, err := getProjects()
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	defer resp.Body.Close()

	projectsBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err, "fatal: error receiving project ID: %s")
	}

	// FIXME: Maybe outsource or implement/import some general query functions for nested structs
	projectsResult := GetProjectsResult{}
	err = json.Unmarshal(projectsBytes, &projectsResult)
	if err != nil {
		util.OutputErrorToConsoleAndExit(err)
	}
	for i := range projectsResult.Projects {
		project := projectsResult.Projects[i]
		if project.Name == projectName {
			projectId = project.ID
			break
		}
	}

	return projectId
}
