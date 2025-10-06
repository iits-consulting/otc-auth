package login

import (
	"context"
	"errors"
	"fmt"

	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/oidc"
	"otc-auth/saml"

	"github.com/golang/glog"
)

func AuthenticateAndGetUnscopedToken(loginCtx context.Context, authInfo common.AuthInfo) error {
	err := config.LoadCloudConfig(authInfo.DomainName)
	if err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}

	if config.IsAuthenticationValid() && !authInfo.OverwriteFile {
		glog.V(1).Info(
			"info: will not retrieve unscoped token, because the current one is still valid.\n" +
				"To overwrite the existing unscoped token, pass the \"--overwrite-token\" argument")
		return nil
	}

	glog.V(1).Info("info: retrieving unscoped token for active cloud...")

	var tokenResponse *common.TokenResponse
	switch authInfo.AuthType {
	case common.AuthTypeIDP:
		switch authInfo.AuthProtocol {
		case common.AuthProtocolSAML:
			tokenResponse, err = saml.AuthenticateAndGetUnscopedToken(loginCtx, authInfo)
			if err != nil {
				return fmt.Errorf("couldn't get unscoped token: %w", err)
			}
		case common.AuthProtocolOIDC:
			tokenResponse, err = oidc.AuthenticateAndGetUnscopedToken(loginCtx, authInfo)
			if err != nil {
				return fmt.Errorf("couldn't get unscoped token: %w", err)
			}
		default:
			return errors.New(
				"fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". " +
					"Please provide a valid argument and try again")
		}
	case common.AuthTypeIAM:
		tokenResponse, err = iam.AuthenticateAndGetUnscopedToken(authInfo)
		if err != nil {
			return fmt.Errorf("couldn't get unscoped token: %w", err)
		}
	default:
		return errors.New(
			"fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". " +
				"Please provide a valid argument and try again")
	}

	if tokenResponse.Token.Secret == "" {
		return errors.New("authorization did not succeed. please try again")
	}
	updateOTCInfoFile(*tokenResponse, authInfo.Region)
	createScopedTokenForEveryProject()
	glog.V(1).Info("info: successfully obtained unscoped token!")
	return nil
}

func createScopedTokenForEveryProject() {
	projectsInActiveCloud := iam.GetProjectsInActiveCloud()
	iam.CreateScopedTokenForEveryProject(projectsInActiveCloud.GetProjectNames())
}

func updateOTCInfoFile(tokenResponse common.TokenResponse, regionCode string) {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		common.ThrowError(err)
	}
	if activeCloud.Domain.Name != tokenResponse.Token.User.Domain.Name {
		// Sanity check: we're in the same cloud as the active cloud
		common.ThrowError(errors.New("fatal: authorization made for wrong cloud configuration"))
	}
	activeCloud.Domain.ID = tokenResponse.Token.User.Domain.ID
	if activeCloud.Username != tokenResponse.Token.User.Name {
		for i, project := range activeCloud.Projects {
			activeCloud.Projects[i].ScopedToken = project.ScopedToken.UpdateToken(config.Token{
				Secret:    "",
				IssuedAt:  "",
				ExpiresAt: "",
			})
		}
	}
	activeCloud.Username = tokenResponse.Token.User.Name
	token := config.Token{
		Secret:    tokenResponse.Token.Secret,
		IssuedAt:  tokenResponse.Token.IssuedAt,
		ExpiresAt: tokenResponse.Token.ExpiresAt,
	}
	activeCloud.Region = regionCode
	activeCloud.UnscopedToken = token
	config.UpdateCloudConfig(*activeCloud)
}
