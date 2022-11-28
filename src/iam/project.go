package iam

import (
	"encoding/json"
	"github.com/go-http-utils/headers"
	"io"
	"net/http"
	"otc-auth/src/common"
	"otc-auth/src/common/endpoints"
	"otc-auth/src/common/headervalues"
)

func getProjectId(projectName string) string {
	projectId := ""
	resp, err := getProjects()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	defer resp.Body.Close()

	projectsBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: error receiving project ID: %s")
	}

	// FIXME: Maybe outsource or implement/import some general query functions for nested structs
	projectsResult := common.GetProjectsResult{}
	err = json.Unmarshal(projectsBytes, &projectsResult)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
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

func getProjects() (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodGet, endpoints.IamProjects, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add(headers.ContentType, headervalues.ApplicationJson)

	client := common.GetHttpClientWithUnscopedToken()
	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, err
}
