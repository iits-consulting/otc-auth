package iam

import (
	"encoding/json"
	"log"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/projects"
)

func GetProjectsInActiveCloud() config.Projects {
	projectsResponse := getProjectsFromServiceProvider()
	var cloudProjects config.Projects
	for _, project := range projectsResponse.Projects {
		cloudProjects = append(cloudProjects, config.Project{
			NameAndIDResource: config.NameAndIDResource{Name: project.Name, ID: project.ID},
		})
	}

	config.UpdateProjects(cloudProjects)
	log.Printf("Projects for active cloud:\n%s \n", strings.Join(cloudProjects.GetProjectNames(), ",\n"))
	return cloudProjects
}

func CreateScopedTokenForEveryProject(projectNames []string) {
	for _, projectName := range projectNames {
		GetProjectScopedToken(projectName)
	}
}

func getServiceClientForActiveCloud() (*golangsdk.ServiceClient, error) {
	cloud := config.GetActiveCloudConfig()
	domain := cloud.Domain

	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region) + "/v3",
		DomainName:       domain.Name,
		TokenID:          cloud.UnscopedToken.Secret,
	})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	return client, err
}

func getProjectsFromServiceProvider() (projectsResponse common.ProjectsResponse) {
	client, err := getServiceClientForActiveCloud()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	projectsList, err := projects.List(client, projects.ListOpts{}).AllPages()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	projectsResponseMap := projectsList.GetBody()

	err = json.Unmarshal(projectsResponseMap, &projectsResponse)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	return projectsResponse
}
