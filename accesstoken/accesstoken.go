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
		// A 404 error is thrown when trying to create a permanent AK/SK when logged in with OIDC or SAML
		var notFound golangsdk.ErrDefault404
		if errors.As(err, &notFound) &&
			strings.Contains(notFound.URL, "OS-CREDENTIAL/credentials") &&
			strings.Contains(string(notFound.Body), "Could not find user:") {
			common.OutputErrorMessageToConsoleAndExit("fatal: cannot create permanent access token when logged in via OIDC or SAML.")
		} else {
			common.OutputErrorToConsoleAndExit(err)
		}
	}
	makeAccessFile(resp, nil)
}

func makeAccessFile(resp *credentials.Credential, tempResp *credentials.TemporaryCredential) {
	if resp == nil && tempResp == nil {
		common.OutputErrorMessageToConsoleAndExit("fatal: no temporary or permanent access keys to write")
	}
	var accessKeyFileContent string
	if resp != nil {
		accessKeyFileContent = fmt.Sprintf(
			"export OS_ACCESS_KEY=%s\n"+
				"export AWS_ACCESS_KEY_ID=%s\n"+
				"export OS_SECRET_KEY=%s\n"+
				"export AWS_SECRET_ACCESS_KEY=%s",
			resp.AccessKey,
			resp.AccessKey,
			resp.SecretKey,
			resp.SecretKey)
	} else {
		accessKeyFileContent = fmt.Sprintf(
			"export OS_ACCESS_KEY=%s\n"+
				"export AWS_ACCESS_KEY_ID=%s\n"+
				"export OS_SECRET_KEY=%s\n"+
				"export AWS_SECRET_ACCESS_KEY=%s\n"+
				"export OS_SESSION_TOKEN=%s\n"+
				"export AWS_SESSION_TOKEN=%s",
			tempResp.AccessKey,
			tempResp.AccessKey,
			tempResp.SecretKey,
			tempResp.SecretKey,
			tempResp.SecurityToken,
			tempResp.SecurityToken)
	}

	common.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)
	log.Println("Access token file created successfully.")
	log.Println("Please source the ak-sk-env.sh file in the current directory manually")
}

func CreateTemporaryAccessToken(durationSeconds int) {
	log.Println("Creating temporary access token file with GTC...")
	resp, err := getTempAccessTokenFromServiceProvider(durationSeconds)
	if err != nil {
		common.OutputErrorToConsoleAndExit(err)
	}

	makeAccessFile(nil, resp)
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

func getTempAccessTokenFromServiceProvider(durationSeconds int) (*credentials.TemporaryCredential, error) {
	client, err := getIdentityServiceClient()
	if err != nil {
		return nil, err
	}
	tempCreds, err := credentials.CreateTemporary(client, credentials.CreateTemporaryOpts{
		Methods:  []string{"token"},
		Duration: durationSeconds,
	}).Extract()
	if err != nil {
		return nil, err
	}
	log.Printf("warning: access key will only be valid until: %v (UTC)", tempCreds.ExpiresAt)
	return tempCreds, err
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
	credResp := credentials.Create(client, credentials.CreateOpts{
		UserID:      user.ID,
		Description: tokenDescription,
	})
	credential, err := credResp.Extract()
	if err != nil {
		credential, err = handlePotentialLimitError(err, user, client, tokenDescription)
	}
	return credential, err
}

func handlePotentialLimitError(err error,
	user *tokens.User,
	client *golangsdk.ServiceClient,
	tokenDescription string,
) (*credentials.Credential, error) {
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
	return nil, err
}

// Replaces AK/SKs made by otc-auth if their descriptions match the default.
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
