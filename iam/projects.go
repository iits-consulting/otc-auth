package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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
	return getProjectsAndPrint(os.Stdout, getProjectsFromServiceProvider, config.UpdateProjects)
}

// getProjectsAndPrint is the testable seam for GetProjectsInActiveCloud.
// fetch retrieves the raw response, update persists projects to disk, and w
// receives the user-facing list. Splitting these dependencies lets us assert
// the list reaches stdout without an HTTP/disk round-trip.
func getProjectsAndPrint(
	w io.Writer,
	fetch func() common.ProjectsResponse,
	update func(config.Projects),
) config.Projects {
	projectsResponse := fetch()
	var cloudProjects config.Projects
	for _, project := range projectsResponse.Projects {
		cloudProjects = append(cloudProjects, config.Project{
			NameAndIDResource: config.NameAndIDResource{Name: project.Name, ID: project.ID},
		})
	}

	update(cloudProjects)
	if err := writeProjectNames(w, cloudProjects); err != nil {
		common.ThrowError(err)
	}
	return cloudProjects
}

func writeProjectNames(w io.Writer, cloudProjects config.Projects) error {
	_, err := fmt.Fprintln(w, strings.Join(cloudProjects.GetProjectNames(), "\n"))
	return err
}

func CreateScopedTokenForEveryProject(projectNames []string) {
	for _, projectName := range projectNames {
		GetScopedToken(projectName)
	}
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
