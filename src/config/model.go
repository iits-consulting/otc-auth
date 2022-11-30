package config

import (
	"errors"
	"fmt"
	"otc-auth/src/common"
	"time"
)

const (
	Unscoped = "unscoped"
	Scoped   = "scoped"
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

func (clouds *Clouds) GetCloudByName(name string) *Cloud {
	for _, cloud := range *clouds {
		if cloud.Domain.Name == name {
			return &cloud
		}
	}
	return nil
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

func (clouds *Clouds) GetActiveCloud() *Cloud {
	cloud, _, err := clouds.FindActiveCloudConfigOrNil()
	if err != nil || cloud == nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: invalid state %s")
	}

	return cloud
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
	Domain   NameAndIdResource `json:"domain"`
	Tokens   Tokens            `json:"tokens"`
	Projects Projects          `json:"projects"`
	Clusters Clusters          `json:"clusters"`
	Username string            `json:"username"`
	Active   bool              `json:"active"`
}

type Project NameAndIdResource
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
		errorMessage := fmt.Sprintf("fatal: project with name %s not found.\n\nUse the cce list-projects command to get a list of projects.", name)
		common.OutputErrorToConsoleAndExit(errors.New(errorMessage))
	}
	return *project
}

func (projects Projects) GetProjectNames() (names []string) {
	for _, project := range projects {
		names = append(names, project.Name)
	}
	return names
}

type Cluster NameAndIdResource
type Clusters []Cluster

func (clusters Clusters) GetClusterNames() (names []string) {
	for _, cluster := range clusters {
		names = append(names, cluster.Name)
	}
	return names
}

func (clusters Clusters) GetClusterByNameOrThrow(name string) Cluster {
	cluster := clusters.FindClusterByName(name)
	if cluster == nil {
		errorMessage := fmt.Sprintf("fatal: cluster with name %s not found.\nuse the cce list-clusters command to retrieve a list of clusters.", name)
		common.OutputErrorToConsoleAndExit(errors.New(errorMessage))
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
	if clusters.FindClusterByName(name) == nil {
		return false
	} else {
		return true
	}
}

type NameAndIdResource struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

type Token struct {
	Type      string `json:"type"`
	Secret    string `json:"secret"`
	IssuedAt  string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
}

type Tokens []Token

func (tokens *Tokens) GetUnscopedToken() Token {
	token, err := tokens.getTokenByType(Unscoped)
	if err != nil || token == nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: no unscoped token found.")
	}
	return *token
}

func (tokens *Tokens) GetScopedToken() Token {
	token, err := tokens.getTokenByType(Scoped)
	if err != nil || token == nil {
		common.OutputErrorToConsoleAndExit(err, "fatal: no scoped token found.")
	}
	return *token
}

func (tokens *Tokens) HasScopedToken() bool {
	if _, err := tokens.getTokenByType(Scoped); err == nil {
		return true
	} else {
		return false
	}
}

func (tokens *Tokens) HasUnscopedToken() bool {
	if _, err := tokens.getTokenByType(Unscoped); err == nil {
		return true
	} else {
		return false
	}
}

func (tokens *Tokens) IsUnscopedTokenValid() bool {
	if common.ParseTimeOrThrow(tokens.GetUnscopedToken().ExpiresAt).After(time.Now()) {
		return true
	} else {
		return false
	}
}

func (tokens *Tokens) IsScopedTokenValid() bool {
	if common.ParseTimeOrThrow(tokens.GetScopedToken().ExpiresAt).After(time.Now()) {
		return true
	} else {
		return false
	}
}

func (tokens *Tokens) UpdateToken(updatedToken Token) (ok bool) {
	if updatedToken.Type == Scoped {
		if tokens.HasScopedToken() {
			scopedToken := &(*tokens)[*tokens.getScopedTokenIndex()]
			scopedToken.Secret = updatedToken.Secret
			scopedToken.ExpiresAt = updatedToken.ExpiresAt
			scopedToken.IssuedAt = updatedToken.IssuedAt

			return true
		}
		*tokens = append(*tokens, updatedToken)
		return true
	} else if updatedToken.Type == Unscoped {
		if tokens.HasUnscopedToken() {
			unscopedToken := &(*tokens)[*tokens.getUnscopedTokenIndex()]
			unscopedToken.Secret = updatedToken.Secret
			unscopedToken.ExpiresAt = updatedToken.ExpiresAt
			unscopedToken.IssuedAt = updatedToken.IssuedAt

			return true
		}
		*tokens = append(*tokens, updatedToken)
		return true
	}
	return false
}

func (tokens *Tokens) getTokenIndex(tokenType string) *int {
	for index, token := range *tokens {
		if token.Type == tokenType {
			return &index
		}
	}
	return nil
}

func (tokens *Tokens) getUnscopedTokenIndex() *int {
	return tokens.getTokenIndex(Unscoped)
}

func (tokens *Tokens) getScopedTokenIndex() *int {
	return tokens.getTokenIndex(Scoped)
}

func (tokens *Tokens) getTokenByType(tokenType string) (*Token, error) {
	for _, token := range *tokens {
		if token.Type == tokenType {
			return &token, nil
		}
	}
	return nil, errors.New("no token found")
}
