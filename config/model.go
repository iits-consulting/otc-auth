package config

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"otc-auth/common"

	"github.com/golang/glog"
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
		return nil, nil, errors.New("more than one cloud active")
	}

	for index, cloud := range *clouds {
		if cloud.Active {
			return &cloud, &index, err
		}
	}

	return nil, nil, errors.New("no active cloud")
}

func (clouds *Clouds) GetActiveCloudIndex() (*int, error) {
	cloud, index, err := clouds.FindActiveCloudConfigOrNil()
	if err != nil || cloud == nil || index == nil {
		return nil, fmt.Errorf("fatal: invalid state %w", err)
	}

	return index, nil
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

func (projects Projects) GetProjectByName(name string) (*Project, error) {
	project := projects.FindProjectByName(name)
	if project == nil {
		return nil, fmt.Errorf(
			"fatal: project with name %s not found.\n\nUse the cce list-projects command to "+
				"get a list of projects", name)
	}
	return project, nil
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

func (clusters Clusters) GetClusterByName(name string) (*Cluster, error) {
	cluster := clusters.FindClusterByName(name)
	if cluster == nil {
		return nil, fmt.Errorf("cluster not found.\nhere's a list of valid clusters:\n%s",
			strings.Join(clusters.GetClusterNames(), ",\n"))
	}
	return cluster, nil
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

func (token *Token) IsValid() bool {
	if token.Secret == "" {
		return false
	}
	timePTR, err := common.ParseTime(token.ExpiresAt)
	if err != nil {
		glog.Warningf("couldn't parse token expires_at: %s", err)
		return false
	}
	return timePTR.After(time.Now())
}

func (token *Token) UpdateToken(updatedToken Token) Token {
	token.Secret = updatedToken.Secret
	token.ExpiresAt = updatedToken.ExpiresAt
	token.IssuedAt = updatedToken.IssuedAt
	return *token
}
