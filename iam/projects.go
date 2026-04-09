package iam

import (
	"encoding/json"
	"errors"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	"github.com/golang/glog"
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
	glog.V(common.InfoLogLevel).Infof("info: projects for active cloud:\n%s \n",
		strings.Join(cloudProjects.GetProjectNames(), ",\n"))
	return cloudProjects
}

func CreateScopedTokenForEveryProject(projectNames []string) error {
	var tokenError error
	store := NewFileConfigStore()
	tc := NewGopherTokenCreator()
	for _, projectName := range projectNames {
		_, err := GetScopedToken(store, tc, projectName) // Getting tokens also caches them for later use
		if err != nil {
			return errors.Join(tokenError, err)
		}
	}
	return nil
}

func getProjectsFromServiceProvider() (projectsResponse common.ProjectsResponse) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	glog.V(common.InfoLogLevel).Infof("info: fetching projects for cloud %s \n", activeCloud.Domain.Name)

	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		DomainID:         activeCloud.Domain.ID,
		TokenID:          activeCloud.UnscopedToken.Secret,
	})
	if err != nil {
		common.ThrowError(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.ThrowError(err)
	}
	projectsList, err := projects.List(client, projects.ListOpts{}).AllPages()
	if err != nil {
		common.ThrowError(err)
	}

	projectsResponseMap := projectsList.GetBody()

	err = json.Unmarshal(projectsResponseMap, &projectsResponse)
	if err != nil {
		common.ThrowError(err)
	}

	return projectsResponse
}
