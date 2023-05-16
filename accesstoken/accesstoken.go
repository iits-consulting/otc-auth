package accesstoken

import (
	"fmt"
	"github.com/go-http-utils/headers"
	golangsdk "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
	"net/http"
	"otc-auth/common"
	"otc-auth/common/endpoints"
	"otc-auth/common/headervalues"
	"otc-auth/common/xheaders"
	"otc-auth/config"
	"strconv"
	"strings"
)

func CreateAccessToken(durationSeconds int, args ...bool) {
	if len(args) > 0 {
		// Use GTC?
		if args[0] {
			println("Creating access token file with GTC...")
			resp, err := getAccessTokenFromServiceProviderGTC(durationSeconds)
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
			return
		}
	}
	println("Creating access token file...")

	response := getAccessTokenFromServiceProvider(strconv.Itoa(durationSeconds))
	bodyBytes := common.GetBodyBytesFromResponse(response)

	accessTokenCreationResponse := common.DeserializeJsonForType[TokenCreationResponse](bodyBytes)

	accessKeyFileContent := fmt.Sprintf(
		"export OS_ACCESS_KEY=%s\n"+
			"export AWS_ACCESS_KEY_ID=%s\n"+
			"export OS_SECRET_KEY=%s\n"+
			"export AWS_SECRET_ACCESS_KEY=%s",
		accessTokenCreationResponse.Credential.Access,
		accessTokenCreationResponse.Credential.Access,
		accessTokenCreationResponse.Credential.Secret,
		accessTokenCreationResponse.Credential.Secret)

	common.WriteStringToFile("./ak-sk-env.sh", accessKeyFileContent)

	println("Access token file created successfully.")
	println("Please source the ak-sk-env.sh file in the current directory manually")
}

func getAccessTokenFromServiceProviderGTC(durationSeconds int) (*credentials.TemporaryCredential, error) {
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

func getAccessTokenFromServiceProvider(durationSeconds string) *http.Response {
	secret := config.GetActiveCloudConfig().UnscopedToken.Secret
	body := fmt.Sprintf("{\"auth\": {\"identity\": {\"methods\": [\"token\"], \"token\": {\"id\": \"%s\", \"duration_seconds\": \"%s\"}}}}", secret, durationSeconds)

	request := common.GetRequest(http.MethodPost, endpoints.IamSecurityTokens, strings.NewReader(body))
	request.Header.Add(headers.ContentType, headervalues.ApplicationJson)
	request.Header.Add(xheaders.XAuthToken, secret)

	return common.HttpClientMakeRequest(request)
}
