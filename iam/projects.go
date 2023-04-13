package iam

import (
	"fmt"
	"github.com/go-http-utils/headers"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"strings"
)

func GetProjectsInActiveCloud() config.Projects {
	projectsResponse := getProjectsFromServiceProvider()
	var projects config.Projects
	for _, project := range projectsResponse.Projects {
		projects = append(projects, config.Project{
			NameAndIdResource: config.NameAndIdResource{Name: project.Name, Id: project.Id},
		})
	}

	config.UpdateProjects(projects)
	println(fmt.Sprintf("Projects for active cloud:\n%s", strings.Join(projects.GetProjectNames(), ",\n")))
	return projects
}

func CreateScopedTokenForEveryProject(projectNames []string) {
	for _, projectName := range projectNames {
		GetScopedToken(projectName)
	}
}

func getProjectsFromServiceProvider() (projectsResponse common.ProjectsResponse) {
	cloud := config.GetActiveCloudConfig()
	println(fmt.Sprintf("info: fetching projects for cloud %s", cloud.Domain.Name))

	request := common.GetRequest(http.MethodGet, endpoints.IamProjects, nil)
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	request.Header.Add(xheaders.XAuthToken, cloud.UnscopedToken.Secret)

	response := common.HttpClientMakeRequest(request)
	bodyBytes := common.GetBodyBytesFromResponse(response)
	projectsResponse = *common.DeserializeJsonForType[common.ProjectsResponse](bodyBytes)

	return projectsResponse
}
