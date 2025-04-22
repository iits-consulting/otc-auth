package login

import (
	"errors"

	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/oidc"
	"otc-auth/saml"

	"github.com/golang/glog"
)

const (
	protocolSAML = "saml"
	protocolOIDC = "oidc"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo, skipTLS bool) {
	err := config.LoadCloudConfig(authInfo.DomainName)
	if err != nil {
		common.ThrowError(err)
	}

	if config.IsAuthenticationValid() && !authInfo.OverwriteFile {
		glog.V(1).Info(
			"info: will not retrieve unscoped token, because the current one is still valid.\n" +
				"To overwrite the existing unscoped token, pass the \"--overwrite-token\" argument")
		return
	}

	glog.V(1).Info("info: retrieving unscoped token for active cloud...")

	var tokenResponse common.TokenResponse
	switch authInfo.AuthType {
	case "idp":
		switch authInfo.AuthProtocol {
		case protocolSAML:
			tokenResponse = saml.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
		case protocolOIDC:
			tokenResponse = oidc.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
		default:
			common.ThrowError(errors.New(
				"fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". " +
					"Please provide a valid argument and try again"))
		}
	case "iam":
		tokenResponse = iam.AuthenticateAndGetUnscopedToken(authInfo)
	default:
		common.ThrowError(errors.New(
			"fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". " +
				"Please provide a valid argument and try again"))
	}

	if tokenResponse.Token.Secret == "" {
		common.ThrowError(errors.New("authorization did not succeed. please try again"))
	}
	updateOTCInfoFile(tokenResponse, authInfo.Region)
	createScopedTokenForEveryProject()
	glog.V(1).Info("info: successfully obtained unscoped token!")
}

func createScopedTokenForEveryProject() {
	projectsInActiveCloud := iam.GetProjectsInActiveCloud()
	iam.CreateScopedTokenForEveryProject(projectsInActiveCloud.GetProjectNames())
}

func updateOTCInfoFile(tokenResponse common.TokenResponse, regionCode string) {
	cloud := config.GetActiveCloudConfig()
	if cloud.Domain.Name != tokenResponse.Token.User.Domain.Name {
		// Sanity check: we're in the same cloud as the active cloud
		common.ThrowError(errors.New("fatal: authorization made for wrong cloud configuration"))
	}
	cloud.Domain.ID = tokenResponse.Token.User.Domain.ID
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
	cloud.Region = regionCode
	cloud.UnscopedToken = token
	config.UpdateCloudConfig(cloud)
}
