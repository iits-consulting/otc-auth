package iam

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	authOpts := golangsdk.AuthOptions{
		DomainName:       authInfo.DomainName,
		Username:         authInfo.Username,
		Password:         authInfo.Password,
		IdentityEndpoint: endpoints.BaseURLIam(authInfo.Region) + "/v3",

		Passcode: authInfo.Otp,
		UserID:   authInfo.UserDomainID,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	tokenResult := tokens.Create(client, &authOpts)

	var tokenMarshalledResult common.TokenResponse
	err = json.Unmarshal(tokenResult.Body, &tokenMarshalledResult)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	token, err := tokenResult.ExtractToken()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	tokenMarshalledResult.Token.Secret = token.ID
	return tokenMarshalledResult
}

func GetDomainScopedToken() {
	cloud := config.GetActiveCloudConfig()
	if cloud.Domain.ScopedToken.IsTokenValid() {
		token := cloud.Domain.ScopedToken
		tokenExpirationDate := common.ParseTimeOrThrow(token.ExpiresAt)
		if tokenExpirationDate.After(time.Now()) {
			log.Printf("info: scoped token is valid until %s \n", tokenExpirationDate.Format(common.PrintTimeFormat))
		}
	}

	log.Println("attempting to request a domain-level scoped token.")
	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region) + "/v3",
		TokenID:          cloud.UnscopedToken.Secret,
		DomainName:       cloud.Domain.Name,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	scopedToken, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	token := config.Token{
		Secret:    scopedToken.ID,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	cloud.Domain.ScopedToken = token
	config.UpdateCloudConfig(cloud)
	log.Println("domain-level scoped token acquired successfully.")
}

func GetProjectScopedToken(projectName string) config.Token {
	project := config.GetActiveCloudConfig().Domain.Projects.GetProjectByNameOrThrow(projectName)
	if project.ScopedToken.IsTokenValid() {
		token := project.ScopedToken

		tokenExpirationDate := common.ParseTimeOrThrow(token.ExpiresAt)
		if tokenExpirationDate.After(time.Now()) {
			log.Printf("info: scoped token is valid until %s \n", tokenExpirationDate.Format(common.PrintTimeFormat))
			return token
		}
	}

	log.Println("attempting to request a project-level scoped token.")
	cloud := getCloudWithProjectScopedTokenFromServiceProvider(projectName)
	config.UpdateCloudConfig(cloud)
	log.Println("project-level scoped token acquired successfully.")
	project = config.GetActiveCloudConfig().Domain.Projects.GetProjectByNameOrThrow(projectName)
	return project.ScopedToken
}

func getCloudWithProjectScopedTokenFromServiceProvider(projectName string) config.Cloud {
	cloud := config.GetActiveCloudConfig()
	projectID := cloud.Domain.Projects.GetProjectByNameOrThrow(projectName).ID

	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region) + "/v3",
		TokenID:          cloud.UnscopedToken.Secret,
		TenantID:         projectID,
		DomainName:       cloud.Domain.Name,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	scopedToken, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	token := config.Token{
		Secret:    scopedToken.ID,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	index := cloud.Domain.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		common.OutputErrorToConsoleAndExit(
			fmt.Errorf("fatal: project with name %s not found.\n"+
				"\nUse the cce list-projects command to get a list of projects",
				projectName))
	}
	cloud.Domain.Projects[*index].ScopedToken = token
	return cloud
}
