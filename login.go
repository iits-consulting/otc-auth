package main

import (
	"fmt"
	"github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/credentials"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack/identity/v3/projects"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/oidc"
	"otc-auth/saml"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) {
	config.LoadCloudConfig(authInfo.DomainName)

	if config.IsAuthenticationValid() && !authInfo.OverwriteFile {
		println("info: will not retrieve unscoped token, because the current one is still valid.\n\nTo overwrite the existing unscoped token, pass the \"--overwrite-token\" argument.")
		return
	}

	println("Retrieving unscoped token for active cloud...")

	// TODO remove
	var tokenResponse common.TokenResponse
	switch authInfo.AuthType {
	case "idp":
		if authInfo.AuthProtocol == protocolSAML {
			tokenResponse = saml.AuthenticateAndGetUnscopedToken(authInfo)
		} else if authInfo.AuthProtocol == protocolOIDC {
			tokenResponse = oidc.AuthenticateAndGetUnscopedToken(authInfo)
			fmt.Println("[*] ", tokenResponse)
			opts := golangsdk.AuthOptions{
				IdentityEndpoint: "https://iam.eu-de.otc.t-systems.com:443/v3",
				DomainID:         "OTC-EU-DE-00000000001000055571",
				TenantID:         "d32336fe26ec415caa04e17e9adfb6a8",
				TokenID:          tokenResponse.Token.Secret,
			}
			client, err := openstack.AuthenticatedClient(opts)
			fmt.Println(client, err)

			listProjects(client)
			// TODO remove
			//tokenResponse2 = gophertelekomcloud2.TestLogin(authInfo)
			// fmt.Println(tokenResponse, tokenResponse2, tokenResponse == tokenResponse2)
		} else {
			common.OutputErrorMessageToConsoleAndExit("fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". Please provide a valid argument and try again.")
		}
	case "iam":
		tokenResponse = iam.AuthenticateAndGetUnscopedToken(authInfo)
	default:
		common.OutputErrorMessageToConsoleAndExit("fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". Please provide a valid argument and try again.")

	}

	if tokenResponse.Token.Secret == "" {
		common.OutputErrorMessageToConsoleAndExit("Authorization did not succeed. Please try again.")
	}
	updateOTCInfoFile(tokenResponse)
	createScopedTokenForEveryProject()
	println("Successfully obtained unscoped token!")
}

type ProjectShortDescription struct {
	description string
	domainId    string
	enabled     bool
	id          string
	isDomain    bool
	links       []ProjectLink
	name        string
	parentId    string
}

type ProjectLink struct {
	description string
	link        string
}

func listProjects(provider *golangsdk.ProviderClient) ([]projects.Project, error) {
	client, err := openstack.NewIdentityV3(provider, golangsdk.EndpointOpts{})
	if err != nil {
		return nil, err
	}

	pager, err := projects.List(client, projects.ListOpts{}).AllPages()
	var projectList []projects.Project

	if err != nil {
		fmt.Println(err)
	}

	prts, err := pager.GetBodyAsMap()

	prtsI := prts["projects"]

	prtsS := prtsI.([]interface{})

	var arr []ProjectShortDescription

	for s := range prtsS {
		proj := prtsS[s].(map[string]interface{})
		projS := ProjectShortDescription{}
		for i, j := range proj {
			switch i {
			case "description":
				projS.description = j.(string)
			case "domain_id":
				projS.domainId = j.(string)
			case "enabled":
				projS.enabled = j.(bool)
			case "id":
				projS.id = j.(string)
			case "is_domain":
				projS.isDomain = j.(bool)
			case "links":
				var arr2 []ProjectLink
				for u := range j.(map[string]interface{}) {
					val := j.(map[string]interface{})[u]
					if val != nil {
						k := ProjectLink{}
						k.description = u
						k.link = val.(string)
						arr2 = append(arr2, k)
					}
				}
				projS.links = arr2
			case "name":
				projS.name = j.(string)
			}
		}
		arr = append(arr, projS)
	}

	fmt.Println("aaa", arr)

	// Try to get AK / SK

	creds, err := credentials.CreateTemporary(client, credentials.CreateTemporaryOpts{
		Methods:  []string{"token"},
		Token:    client.Token(),
		Duration: 60,
	}).Extract()

	fmt.Println(creds)

	if err != nil {
		fmt.Println(err)
	}

	if err != nil {
		return nil, err
	}

	return projectList, nil
}

func createScopedTokenForEveryProject() {
	projectsInActiveCloud := iam.GetProjectsInActiveCloud()
	iam.CreateScopedTokenForEveryProject(projectsInActiveCloud.GetProjectNames())
}

func updateOTCInfoFile(tokenResponse common.TokenResponse) {
	cloud := config.GetActiveCloudConfig()
	if cloud.Domain.Name != tokenResponse.Token.User.Domain.Name {
		// Sanity check: we're in the same cloud as the active cloud
		common.OutputErrorMessageToConsoleAndExit("fatal: authorization made for wrong cloud configuration")
	}
	cloud.Domain.Id = tokenResponse.Token.User.Domain.Id
	if cloud.Username != tokenResponse.Token.User.Name {
		for i, project := range cloud.Projects {
			cloud.Projects[i].ScopedToken = project.ScopedToken.UpdateToken(config.Token{
				Secret:    "",
				IssuedAt:  "",
				ExpiresAt: "",
			})
		}
	}
	cloud.Username = tokenResponse.Token.User.Name
	token := config.Token{
		Secret:    tokenResponse.Token.Secret,
		IssuedAt:  tokenResponse.Token.IssuedAt,
		ExpiresAt: tokenResponse.Token.ExpiresAt,
	}

	cloud.UnscopedToken = token
	config.UpdateCloudConfig(cloud)
}
