package iam

import (
	"encoding/json"
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
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get openstack client: %w", err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
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

func GetScopedToken(projectName string) (*config.Token, error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get active cloud config: %w", err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get project named '%s': %w", projectName, err)
	}
	if project.ScopedToken.IsTokenValid() {
		tokenExpirationDate, parseErr := common.ParseTime(project.ScopedToken.ExpiresAt)
		if parseErr != nil {
			return nil, fmt.Errorf("couldn't parse token expiry time: %w", err)
		}
		if tokenExpirationDate.After(time.Now()) {
			glog.V(common.InfoLogLevel).Infof("info: scoped token is valid until %s \n",
				tokenExpirationDate.Format(common.PrintTimeFormat))
			return &project.ScopedToken, nil
		}
	}

	glog.V(common.InfoLogLevel).Infof("info: attempting to request a scoped token for %s\n", projectName)
	cloud, err := getCloudWithScopedTokenFromServiceProvider(projectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get token from sp: %w", err)
	}
	config.UpdateCloudConfig(*cloud)
	glog.V(common.InfoLogLevel).Info("info: scoped token acquired successfully")
	project, err = activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get project by named '%s': %w", projectName, err)
	}
	return &project.ScopedToken, nil
}

// TODO - DRY
func getCloudWithScopedTokenFromServiceProvider(projectName string) (*config.Cloud, error) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		return nil, fmt.Errorf("couldn't get active cloud config: %w", err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		return nil, fmt.Errorf("couldn't get project by named '%s': %w", projectName, err)
	}

	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		TokenID:          activeCloud.UnscopedToken.Secret,
		TenantID:         project.ID,
		DomainName:       activeCloud.Domain.Name,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, fmt.Errorf("couldn't get authed client: %w", err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, fmt.Errorf("couldn't get identity client: %w", err)
	}

	scopedToken, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		return nil, fmt.Errorf("couldn't create and extract token: %w", err)
	}

	token := config.Token{
		Secret:    scopedToken.ID,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	index := activeCloud.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		return nil, fmt.Errorf(
			"fatal: project with name %s not found.\n"+
				"\nUse the cce list-projects command to get a list of projects",
			projectName)
	}
	activeCloud.Projects[*index].ScopedToken = token
	return activeCloud, nil
}
