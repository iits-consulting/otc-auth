package iam

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	"github.com/golang/glog"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) (*common.TokenResponse, error) {
	authOpts := golangsdk.AuthOptions{
		DomainName:       authInfo.DomainName,
		Username:         authInfo.Username,
		Password:         authInfo.Password,
		IdentityEndpoint: endpoints.BaseURLIam(authInfo.Region),

		Passcode: authInfo.Otp,
		UserID:   authInfo.UserID,
	}

	client, err := newIdentityV3Client(authOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get identity client: %w", err)
	}

	tokenResult := tokens.Create(client, &authOpts)

	var tokenMarshalledResult common.TokenResponse
	err = json.Unmarshal(tokenResult.Body, &tokenMarshalledResult)
	if err != nil {
		return nil, fmt.Errorf("couldn't unmarshal token: %w", err)
	}

	token, err := tokenResult.ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("couldn't extract token: %w", err)
	}
	tokenMarshalledResult.Token.Secret = token.ID
	return &tokenMarshalledResult, nil
}

type TokenCreator interface {
	CreateToken(opts golangsdk.AuthOptions) (*tokens.Token, error)
}

type gopherTokenCreator struct{}

func NewGopherTokenCreator() TokenCreator {
	return &gopherTokenCreator{}
}

type ConfigStore interface {
	GetActiveCloud() (*config.Cloud, error)
	SaveActiveCloud(config.Cloud) error
}

type fileConfigStore struct{}

func NewFileConfigStore() ConfigStore {
	return &fileConfigStore{}
}

func (s *fileConfigStore) GetActiveCloud() (*config.Cloud, error) {
	return config.GetActiveCloudConfig()
}

func (s *fileConfigStore) SaveActiveCloud(cloud config.Cloud) error {
	return config.UpdateCloudConfig(cloud)
}

func (g *gopherTokenCreator) CreateToken(authOpts golangsdk.AuthOptions) (*tokens.Token, error) {
	client, err := newIdentityV3Client(authOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get identity client: %w", err)
	}

	token, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("couldn't create and extract token: %w", err)
	}
	return token, nil
}

func newIdentityV3Client(authOpts golangsdk.AuthOptions) (*golangsdk.ServiceClient, error) {
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get authenticated client: %w", err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get identity client: %w", err)
	}
	return client, nil
}

func GetScopedToken(
	store ConfigStore,
	tokenCreator TokenCreator,
	projectName string,
) (*config.Token, error) {
	activeCloud, err := store.GetActiveCloud()
	if err != nil {
		return nil, fmt.Errorf("couldn't get active cloud config: %w", err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get project named '%s': %w", projectName, err)
	}

	if project.ScopedToken.IsValid() {
		glog.V(common.InfoLogLevel).Infof("scoped token is valid until %s", project.ScopedToken.ExpiresAt)
		return &project.ScopedToken, nil
	}

	glog.V(common.InfoLogLevel).Infof("attempting to refresh scoped token for %s", projectName)

	newToken, err := fetchNewScopedToken(
		tokenCreator,
		activeCloud.UnscopedToken.Secret,
		project.ID,
		activeCloud.Region,
		activeCloud.Domain.Name,
	)
	if err != nil {
		return nil, fmt.Errorf("couldn't get new scoped token from provider: %w", err)
	}

	index := activeCloud.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		return nil, fmt.Errorf("could not find project index for %s", projectName)
	}
	activeCloud.Projects[*index].ScopedToken = *newToken

	err = store.SaveActiveCloud(*activeCloud)
	if err != nil {
		return nil, fmt.Errorf("couldn't save active cloud: %w", err)
	}

	glog.Info("scoped token acquired and saved successfully")
	return newToken, nil
}

func fetchNewScopedToken(tc TokenCreator, unscopedToken, projectID, region, domainName string) (*config.Token, error) {
	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(region),
		TokenID:          unscopedToken,
		TenantID:         projectID,
		DomainName:       domainName,
	}

	gopherToken, err := tc.CreateToken(authOpts)
	if err != nil {
		return nil, err
	}

	return gopherTokenToConfigToken(gopherToken)
}

func gopherTokenToConfigToken(gopherToken *tokens.Token) (*config.Token, error) {
	if gopherToken == nil {
		return nil, errors.New("token to convert is nil")
	}
	return &config.Token{
		Secret:    gopherToken.ID,
		ExpiresAt: gopherToken.ExpiresAt.Format(time.RFC3339),
	}, nil
}

func configTokenToGopherToken(configToken *config.Token) (*tokens.Token, error) {
	if configToken == nil {
		return nil, errors.New("token to convert is nil")
	}
	parse, err := time.Parse(time.RFC3339, configToken.ExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse expiry time: %w", err)
	}
	return &tokens.Token{
		ID:        configToken.Secret,
		ExpiresAt: parse,
	}, nil
}
