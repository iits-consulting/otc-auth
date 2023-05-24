package iam

import (
	"errors"
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
	"time"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) (tokenResponse common.TokenResponse) {
	authOpts := golangsdk.AuthOptions{
		DomainName:       authInfo.DomainName,
		Username:         authInfo.Username,
		Password:         authInfo.Password,
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3"}

	if authInfo.Otp != "" && authInfo.UserDomainId != "" {
		// TODO
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	token, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	user, err := tokens.Create(client, &authOpts).ExtractUser()
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	tokenResponse.Token.Secret = token.ID
	tokenResponse.Token.ExpiresAt = token.ExpiresAt.Format(time.RFC3339)
	tokenResponse.Token.User.Domain.Id = user.Domain.ID
	tokenResponse.Token.User.Domain.Name = user.Domain.Name
	tokenResponse.Token.User.Name = user.Name
	// TODO time issued?? Is this used?
	return tokenResponse
}

func GetScopedToken(projectName string) config.Token {
	project := config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)

	if project.ScopedToken.IsTokenValid() {
		token := project.ScopedToken

		tokenExpirationDate := common.ParseTimeOrThrow(token.ExpiresAt)
		if tokenExpirationDate.After(time.Now()) {
			println(fmt.Sprintf("info: scoped token is valid until %s", tokenExpirationDate.Format(common.PrintTimeFormat)))
			return token
		}
	}

	println("attempting to request a scoped token.")
	getScopedTokenFromServiceProvider(projectName) // TODO - Seems a little dirty, might want to actually return a value and not have the cloud config updated as a side-effect
	project = config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	return project.ScopedToken
}

func getScopedTokenFromServiceProvider(projectName string) {
	cloud := config.GetActiveCloudConfig()
	projectId := cloud.Projects.GetProjectByNameOrThrow(projectName).Id

	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		TokenID:          cloud.UnscopedToken.Secret,
		TenantID:         projectId,
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
		Secret: scopedToken.ID,
		// TODO IssuedAt:  tokenResponse.Token.IssuedAt,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	index := cloud.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		errorMessage := fmt.Sprintf("fatal: project with name %s not found.\n\nUse the cce list-projects command to get a list of projects.", projectName)
		common.OutputErrorToConsoleAndExit(errors.New(errorMessage))
	}
	cloud.Projects[*index].ScopedToken = token
	config.UpdateCloudConfig(cloud)
	println("scoped token acquired successfully.")
}
