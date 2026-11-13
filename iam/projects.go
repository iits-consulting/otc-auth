package iam

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"otc-auth/common"
	"otc-auth/config"

	"github.com/golang/glog"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/projects"
)

func FetchProjects() config.Projects {
	return projectsFromResponse(getProjectsFromServiceProvider())
}

func projectsFromResponse(projectsResponse common.ProjectsResponse) config.Projects {
	var cloudProjects config.Projects
	for _, project := range projectsResponse.Projects {
		cloudProjects = append(cloudProjects, config.Project{
			NameAndIDResource: config.NameAndIDResource{Name: project.Name, ID: project.ID},
		})
	}
	return cloudProjects
}

func WriteProjectNames(w io.Writer, cloudProjects config.Projects) error {
	_, err := fmt.Fprintln(w, strings.Join(cloudProjects.GetProjectNames(), "\n"))
	return err
}

func CreateScopedTokenForEveryProject(projectNames []string) error {
	return createScopedTokenForEveryProject(NewFileConfigStore(), NewGopherTokenCreator(), projectNames)
}

func createScopedTokenForEveryProject(store ConfigStore, tc TokenCreator, projectNames []string) error {
	tokens, err := fetchScopedTokens(store, tc, projectNames)
	if err != nil {
		return err
	}
	return saveScopedTokens(store, tokens)
}

func fetchScopedTokens(store ConfigStore, tc TokenCreator, projectNames []string) (map[string]config.Token, error) {
	cloud, err := store.GetActiveCloud()
	if err != nil {
		return nil, fmt.Errorf("couldn't get active cloud config: %w", err)
	}

	tokens := make(map[string]config.Token, len(projectNames))
	for _, projectName := range projectNames {
		token, tokenErr := scopedTokenForProject(cloud, tc, projectName)
		if tokenErr != nil {
			return nil, tokenErr
		}
		tokens[projectName] = *token
	}
	return tokens, nil
}

// saveScopedTokens writes all tokens to the active cloud in a single config update.
func saveScopedTokens(store ConfigStore, tokens map[string]config.Token) error {
	if len(tokens) == 0 {
		return nil
	}
	cloud, err := store.GetActiveCloud()
	if err != nil {
		return fmt.Errorf("couldn't get active cloud config: %w", err)
	}

	for projectName, token := range tokens {
		index := cloud.Projects.FindProjectIndexByName(projectName)
		if index == nil {
			return fmt.Errorf("could not find project index for %s", projectName)
		}
		cloud.Projects[*index].ScopedToken = token
	}

	if err = store.SaveActiveCloud(*cloud); err != nil {
		return fmt.Errorf("couldn't save active cloud: %w", err)
	}
	glog.V(common.InfoLogLevel).Infof("saved scoped tokens for %d projects", len(tokens))
	return nil
}

func getProjectsFromServiceProvider() (projectsResponse common.ProjectsResponse) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	glog.V(common.InfoLogLevel).Infof("info: fetching projects for cloud %s \n", activeCloud.Domain.Name)

	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: iamBaseURL(activeCloud.Region),
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
