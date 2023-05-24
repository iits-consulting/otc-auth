package main

import (
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

	var tokenResponse common.TokenResponse
	switch authInfo.AuthType {
	case "idp":
		if authInfo.AuthProtocol == protocolSAML {
			tokenResponse = saml.AuthenticateAndGetUnscopedToken(authInfo)
		} else if authInfo.AuthProtocol == protocolOIDC {
			tokenResponse = oidc.AuthenticateAndGetUnscopedToken(authInfo)
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
