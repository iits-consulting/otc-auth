package iam

import (
	"encoding/json"
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
	"time"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) common.TokenResponse {
	authOpts := golangsdk.AuthOptions{
		DomainName:       authInfo.DomainName,
		Username:         authInfo.Username,
		Password:         authInfo.Password,
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",

		Passcode: authInfo.Otp,
		UserID:   authInfo.UserDomainId,
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
		common.OutputErrorToConsoleAndExit(fmt.Errorf("fatal: project with name %s not found.\n\nUse the cce list-projects command to get a list of projects.", projectName))
	}
	cloud.Projects[*index].ScopedToken = token
	config.UpdateCloudConfig(cloud)
	println("scoped token acquired successfully.")
}
