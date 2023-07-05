package config

import (
	"fmt"
	"time"

	"otc-auth/common"
)

type OtcConfigContent struct {
	Clouds Clouds `json:"clouds"`
}

type Clouds []Cloud

func (clouds *Clouds) ContainsCloud(name string) bool {
	for _, cloud := range *clouds {
		if cloud.Domain.Name == name {
			return true
		}
	}
	return false
}

func (clouds *Clouds) RemoveCloudByNameIfExists(name string) {
	for index, cloud := range *clouds {
		if cloud.Domain.Name == name {
			*clouds = common.RemoveFromSliceAtIndex(*clouds, index)
		}
	}
}

func (clouds *Clouds) SetActiveByName(name string) {
	for index, cloud := range *clouds {
		if cloud.Domain.Name == name {
			(*clouds)[index].Active = true
		} else {
			(*clouds)[index].Active = false
		}
	}
}

func (clouds *Clouds) FindActiveCloudConfigOrNil() (cloud *Cloud, index *int, err error) {
	if clouds.NumberOfActiveCloudConfigs() > 1 {
		return nil, nil, fmt.Errorf("more than one cloud active")
	}

	for index, cloud := range *clouds {
		if cloud.Active {
			return &cloud, &index, err
		}
	}

	return nil, nil, fmt.Errorf("no active cloud")
}

func (clouds *Clouds) GetActiveCloudIndex() int {
	cloud, index, err := clouds.FindActiveCloudConfigOrNil()
	if err != nil || cloud == nil || index == nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: invalid state %s")
	}

	return *index
}

func (clouds *Clouds) NumberOfActiveCloudConfigs() int {
	count := 0
	for _, cloud := range *clouds {
		if cloud.Active {
			count++
		}
	}
	return count
}

type Cloud struct {
	Region        string            `json:"region"`
	Domain        NameAndIDResource `json:"domain"`
	UnscopedToken Token             `json:"unscopedToken"`
	Projects      Projects          `json:"projects"`
	Clusters      Clusters          `json:"clusters"`
	Username      string            `json:"username"`
	Active        bool              `json:"active"`
}

type Project struct {
	NameAndIDResource
	ScopedToken Token `json:"scopedToken"`
}
type Projects []Project

func (projects Projects) FindProjectByName(name string) *Project {
	for _, project := range projects {
		if project.Name == name {
			return &project
		}
	}
	return nil
}

func (projects Projects) GetProjectByNameOrThrow(name string) Project {
	project := projects.FindProjectByName(name)
	if project == nil {
		common.OutputErrorToConsoleAndExit(fmt.Errorf(
			"fatal: project with name %s not found.\n\nUse the cce list-projects command to "+
				"get a list of projects", name))
	}
	return *project
}

func (projects Projects) FindProjectIndexByName(name string) *int {
	for i, project := range projects {
		if project.Name == name {
			return &i
		}
	}
	return nil
}

func (projects Projects) GetProjectNames() []string {
	var names []string
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
}

type (
	Cluster  NameAndIDResource
	Clusters []Cluster
)

func (clusters Clusters) GetClusterNames() []string {
	var names []string
	for _, cluster := range clusters {
		names = append(names, cluster.Name)
	}
	return names
}

func (clusters Clusters) GetClusterByNameOrThrow(name string) Cluster {
	cluster := clusters.FindClusterByName(name)
	if cluster == nil {
		common.OutputErrorToConsoleAndExit(fmt.Errorf(
			"fatal: cluster with name %s not found.\nuse the cce list-clusters command to retrieve "+
				"a list of clusters", name))
	}
	return *cluster
}

func (clusters Clusters) FindClusterByName(name string) *Cluster {
	for _, cluster := range clusters {
		if cluster.Name == name {
			return &cluster
		}
	}
	return nil
}

func (clusters Clusters) ContainsClusterByName(name string) bool {
	return clusters.FindClusterByName(name) != nil
}

type NameAndIDResource struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type Token struct {
	Secret    string `json:"secret"`
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
}

func (token *Token) IsTokenValid() bool {
	return common.ParseTimeOrThrow(token.ExpiresAt).After(time.Now())
}

func (token *Token) UpdateToken(updatedToken Token) Token {
	token.Secret = updatedToken.Secret
	token.ExpiresAt = updatedToken.ExpiresAt
	token.IssuedAt = updatedToken.IssuedAt
	return *token
}
