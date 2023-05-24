package iam

import (
	"encoding/json"
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/projects"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
	"strings"
)

func GetProjectsInActiveCloud() config.Projects {
	projectsResponse := getProjectsFromServiceProvider()
	var cloudProjects config.Projects
	for _, project := range projectsResponse.Projects {
		cloudProjects = append(cloudProjects, config.Project{
			NameAndIdResource: config.NameAndIdResource{Name: project.Name, Id: project.Id},
		})
	}

	config.UpdateProjects(cloudProjects)
	println(fmt.Sprintf("Projects for active cloud:\n%s", strings.Join(cloudProjects.GetProjectNames(), ",\n")))
	return cloudProjects
}

func CreateScopedTokenForEveryProject(projectNames []string) {
	for _, projectName := range projectNames {
		GetScopedToken(projectName)
	}
}

func getProjectsFromServiceProvider() common.ProjectsResponse {
	cloud := config.GetActiveCloudConfig()
	println(fmt.Sprintf("info: fetching projects for cloud %s", cloud.Domain.Name))

	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TokenID:          cloud.UnscopedToken.Secret,
	})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	projectsResponse, err := projects.List(client, projects.ListOpts{}).AllPages()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	projectsResponseMap := projectsResponse.GetBody()
	// forgive me
	var out common.ProjectsResponse
	err = json.Unmarshal(projectsResponseMap, &out)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return out
}
