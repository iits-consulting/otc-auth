package accesstoken

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/config"

	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/tokens"
)

func CreateAccessToken(tokenDescription string) {
	log.Println("Creating access token file with GTC...")
	resp, err := getAccessTokenFromServiceProvider(tokenDescription)
	if err != nil {
		// Handle error currently thrown when logged in by OIDC
		var convErr golangsdk.ErrDefault404
		if errors.As(err, &convErr) {
			if convErr.ErrUnexpectedResponseCode.Actual == 404 &&
				strings.Contains(convErr.ErrUnexpectedResponseCode.URL,
					"OS-CREDENTIAL/credentials") {
				common.OutputErrorMessageToConsoleAndExit(
					"fatal: cannot generate AK/SK if logged in via OIDC")
			}
		}
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
	log.Println("Access token file created successfully.")
	log.Println("Please source the ak-sk-env.sh file in the current directory manually")
}

func ListAccessToken() ([]credentials.Credential, error) {
	client, err := getIdentityServiceClient()
	if err != nil {
		return nil, err
	}
	user, err := tokens.Get(client, config.GetActiveCloudConfig().UnscopedToken.Secret).ExtractUser()
	if err != nil {
		return nil, fmt.Errorf("couldn't get user: %w", err)
	}
	return credentials.List(client, credentials.ListOpts{UserID: user.ID}).Extract()
}

func getAccessTokenFromServiceProvider(tokenDescription string) (*credentials.Credential, error) {
	client, err := getIdentityServiceClient()
	if err != nil {
		return nil, err
	}
	user, err := tokens.Get(client, config.GetActiveCloudConfig().UnscopedToken.Secret).ExtractUser()
	if err != nil {
		return nil, fmt.Errorf("couldn't get user: %w", err)
	}
	credential, err := credentials.Create(client, credentials.CreateOpts{
		UserID:      user.ID,
		Description: tokenDescription,
	}).Extract()

	var badRequest golangsdk.ErrDefault400
	if errors.As(err, &badRequest) {
		accessTokens, listErr := ListAccessToken()
		if listErr != nil {
			return nil, listErr
		}

		//nolint:gomnd // The OpenTelekomCloud only lets users have up to two keys
		if len(accessTokens) == 2 {
			log.Printf("Hit the limit for access keys on OTC. You can only have 2. Removing keys made by otc-auth...")
			return conditionallyReplaceAccessTokens(user, client, tokenDescription, accessTokens)
		}
		return nil, err
	}
	return credential, err
}

// Replaces AK/SKs made by otc-auth if their descriptions match the default..
func conditionallyReplaceAccessTokens(user *tokens.User, client *golangsdk.ServiceClient,
	tokenDescription string, accessTokens []credentials.Credential,
) (*credentials.Credential, error) {
	changed := false
	for _, token := range accessTokens {
		if token.Description == "Token by otc-auth" {
			err := DeleteAccessToken(token.AccessKey)
			if err != nil {
				return nil, err
			}
			changed = true
			break
		}
	}

	if changed {
		return credentials.Create(client, credentials.CreateOpts{
			UserID:      user.ID,
			Description: tokenDescription,
		}).Extract()
	}
	return nil, errors.New("fatal: couldn't find a token created by this tool to replace")
}

func DeleteAccessToken(token string) error {
	client, err := getIdentityServiceClient()
	if err != nil {
		return err
	}
	return credentials.Delete(client, token).ExtractErr()
}

func getIdentityServiceClient() (*golangsdk.ServiceClient, error) {
	provider, err := openstack.AuthenticatedClient(golangsdk.AuthOptions{
		IdentityEndpoint: endpoints.BaseURLIam(config.GetActiveCloudConfig().Region) + "/v3",
		DomainID:         config.GetActiveCloudConfig().Domain.ID,
		TokenID:          config.GetActiveCloudConfig().UnscopedToken.Secret,
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get provider: %w", err)
	}
	return openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
}
