package iam

import (
	"encoding/json"
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
		IdentityEndpoint: endpoints.BaseURLIam(authInfo.Region),

		Passcode: authInfo.Otp,
		UserID:   authInfo.UserDomainID,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatal(err)
	}

	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		log.Fatal(err)
	}

	tokenResult := tokens.Create(client, &authOpts)

	var tokenMarshalledResult common.TokenResponse
	err = json.Unmarshal(tokenResult.Body, &tokenMarshalledResult)
	if err != nil {
		log.Fatal(err)
	}

	token, err := tokenResult.ExtractToken()
	if err != nil {
		log.Fatal(err)
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
			log.Printf("info: scoped token is valid until %s \n", tokenExpirationDate.Format(common.PrintTimeFormat))
			return token
		}
	}

	log.Println("attempting to request a scoped token")
	cloud := getCloudWithScopedTokenFromServiceProvider(projectName)
	config.UpdateCloudConfig(cloud)
	log.Println("scoped token acquired successfully")
	project = config.GetActiveCloudConfig().Projects.GetProjectByNameOrThrow(projectName)
	return project.ScopedToken
}

func getCloudWithScopedTokenFromServiceProvider(projectName string) config.Cloud {
	cloud := config.GetActiveCloudConfig()
	projectID := cloud.Projects.GetProjectByNameOrThrow(projectName).ID

	authOpts := golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(cloud.Region),
		TokenID:          cloud.UnscopedToken.Secret,
		TenantID:         projectID,
		DomainName:       cloud.Domain.Name,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		log.Fatal(err)
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		log.Fatal(err)
	}

	scopedToken, err := tokens.Create(client, &authOpts).ExtractToken()
	if err != nil {
		log.Fatal(err)
	}

	token := config.Token{
		Secret:    scopedToken.ID,
		ExpiresAt: scopedToken.ExpiresAt.Format(time.RFC3339),
	}
	index := cloud.Projects.FindProjectIndexByName(projectName)
	if index == nil {
		log.Fatalf(
			"fatal: project with name %s not found.\n"+
				"\nUse the cce list-projects command to get a list of projects",
			projectName)
	}
	cloud.Projects[*index].ScopedToken = token
	return cloud
}
