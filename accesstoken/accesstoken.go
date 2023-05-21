package accesstoken

import (
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
)

func CreateAccessToken(tokenDescription string) {
	println("Creating access token file with GTC...")
	resp, err := getAccessTokenFromServiceProvider(tokenDescription)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	accessKeyFileContent := fmt.Sprintf(
		"export OS_ACCESS_KEY=%s\n"+
			"export AWS_ACCESS_KEY_ID=%s\n"+
			"export OS_SECRET_KEY=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s",
		resp.AccessKey,
		resp.AccessKey,
		resp.SecretKey,
		resp.SecretKey)

	common.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)
	println("Access token file created successfully.")
	println("Please source the ak-sk-env.sh file in the current directory manually")
}

func ListAccessToken() ([]credentials.Credential, error) {
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TenantID:         config.GetActiveCloudConfig().Domain.Name,
		TokenID:          config.GetActiveCloudConfig().UnscopedToken.Secret,
	})
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	user, err := tokens.Get(client, config.GetActiveCloudConfig().UnscopedToken.Secret).ExtractUser()
	if err != nil {
		return nil, err
	}
	return credentials.List(client, credentials.ListOpts{UserID: user.ID}).Extract()
}

func getAccessTokenFromServiceProvider(tokenDescription string) (*credentials.Credential, error) {
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TenantID:         config.GetActiveCloudConfig().Domain.Name,
		TokenID:          config.GetActiveCloudConfig().UnscopedToken.Secret,
	})
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, err
	}
	user, err := tokens.Get(client, config.GetActiveCloudConfig().UnscopedToken.Secret).ExtractUser()
	if err != nil {
		return nil, err
	}
	return credentials.Create(client, credentials.CreateOpts{
		UserID:      user.ID,
		Description: tokenDescription,
	}).Extract()
}

func DeleteAccessToken(token string) error {
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseUrlIam + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.Id,
		TenantID:         config.GetActiveCloudConfig().Domain.Name,
		TokenID:          config.GetActiveCloudConfig().UnscopedToken.Secret,
	})
	if err != nil {
		return err
	}
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return err
	}
	return credentials.Delete(client, token).ExtractErr()
}
