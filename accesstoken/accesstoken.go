package accesstoken

import (
	"fmt"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"
)

func CreateAccessToken(durationSeconds int) {
	println("Creating access token file with GTC...")
	resp, err := getAccessTokenFromServiceProvider(durationSeconds)
	if err != nil {
		println("[!] ", err)
		return
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

func getAccessTokenFromServiceProvider(durationSeconds int) (*credentials.TemporaryCredential, error) {
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
	return credentials.CreateTemporary(client, credentials.CreateTemporaryOpts{
		Methods:  []string{"token"},
		Token:    config.GetActiveCloudConfig().UnscopedToken.Secret,
		Duration: durationSeconds,
	}).Extract()
}
