package login

import (
	"log"

	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/oidc"
	"otc-auth/saml"
)

const (
	protocolSAML = "saml"
	protocolOIDC = "oidc"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo, skipTLS bool) {
	config.LoadCloudConfig(authInfo.DomainName)

	if config.IsAuthenticationValid() && !authInfo.OverwriteFile {
		log.Println(
			"info: will not retrieve unscoped token, because the current one is still valid.\n" +
				"\nTo overwrite the existing unscoped token, pass the \"--overwrite-token\" argument.")
		return
	}

	log.Println("Retrieving unscoped token for active cloud...")

	var tokenResponse common.TokenResponse
	switch authInfo.AuthType {
	case "idp":
		switch authInfo.AuthProtocol {
		case protocolSAML:
			tokenResponse = saml.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
		case protocolOIDC:
			tokenResponse = oidc.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
		default:
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". " +
					"Please provide a valid argument and try again.")
		}
	case "iam":
		tokenResponse = iam.AuthenticateAndGetUnscopedToken(authInfo)
	default:
		common.OutputErrorMessageToConsoleAndExit(
			"fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". " +
				"Please provide a valid argument and try again.")
	}

	if tokenResponse.Token.Secret == "" {
		common.OutputErrorMessageToConsoleAndExit("Authorization did not succeed. Please try again.")
	}
	updateOTCInfoFile(tokenResponse, authInfo.Region)
	createDomainScopedToken()

	log.Println("Successfully obtained unscoped token!")
}

func createDomainScopedToken() {
	iam.GetDomainScopedToken()
	createScopedTokenForEveryProject()
}

func createScopedTokenForEveryProject() {
	projectsInActiveCloud := iam.GetProjectsInActiveCloud()
	iam.CreateScopedTokenForEveryProject(projectsInActiveCloud.GetProjectNames())
}

func updateOTCInfoFile(tokenResponse common.TokenResponse, regionCode string) {
	cloud := config.GetActiveCloudConfig()
	domain := cloud.Domain
	if domain.Name != tokenResponse.Token.User.Domain.Name {
		// Sanity check: we're in the same cloud as the active cloud
		common.OutputErrorMessageToConsoleAndExit("fatal: authorization made for wrong cloud configuration")
	}
	domain.ID = tokenResponse.Token.User.Domain.ID
	if cloud.Username != tokenResponse.Token.User.Name {
		for i, project := range domain.Projects {
			domain.Projects[i].ScopedToken = project.ScopedToken.UpdateToken(config.Token{
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
	cloud.Region = regionCode
	cloud.UnscopedToken = token
	config.UpdateCloudConfig(cloud)
}
