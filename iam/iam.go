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

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
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
		common.ThrowError(err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.ThrowError(err)
	}

	tokenResult := tokens.Create(client, &authOpts)

	var tokenMarshalledResult common.TokenResponse
	err = json.Unmarshal(tokenResult.Body, &tokenMarshalledResult)
	if err != nil {
		common.ThrowError(err)
	}

	token, err := tokenResult.ExtractToken()
	if err != nil {
		common.ThrowError(err)
	}
	tokenMarshalledResult.Token.Secret = token.ID
	return tokenMarshalledResult
}

func GetScopedToken(projectName string) config.Token {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		common.ThrowError(err)
	}
	if project.ScopedToken.IsTokenValid() {
		token := project.ScopedToken

		tokenExpirationDate, parseErr := common.ParseTime(token.ExpiresAt)
		if parseErr != nil {
			common.ThrowError(parseErr)
		}
		if tokenExpirationDate.After(time.Now()) {
			glog.V(1).Infof("info: scoped token is valid until %s \n", tokenExpirationDate.Format(common.PrintTimeFormat))
			return token
		}
	}

	glog.V(1).Infof("info: attempting to request a scoped token for %s\n", projectName)
	cloud := getCloudWithScopedTokenFromServiceProvider(projectName)
	config.UpdateCloudConfig(cloud)
	glog.V(1).Info("info: scoped token acquired successfully")
	project, err = activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		common.ThrowError(err)
	}
	return project.ScopedToken
}

func getCloudWithScopedTokenFromServiceProvider(projectName string) config.Cloud {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	project, err := activeCloud.Projects.GetProjectByName(projectName)
	if err != nil {
		common.ThrowError(err)
	}

	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(activeCloud.Region),
		TokenID:          activeCloud.UnscopedToken.Secret,
		TenantID:         project.ID,
		DomainName:       activeCloud.Domain.Name,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.ThrowError(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.ThrowError(err)
	}

	scopedToken, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		common.ThrowError(err)
	}

	token := config.Token{
		Secret:    scopedToken.ID,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	index := activeCloud.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		common.ThrowError(fmt.Errorf(
			"fatal: project with name %s not found.\n"+
				"\nUse the projects list command to get a list of projects",
			projectName))
	}
	activeCloud.Projects[*index].ScopedToken = token
	return *activeCloud
}
