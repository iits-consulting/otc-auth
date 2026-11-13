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

type tokenProviderFunc func(context.Context, common.AuthInfo) (*common.TokenResponse, error)

func getTokenProvider(authInfo common.AuthInfo) (tokenProviderFunc, error) {
	switch authInfo.AuthType {
	case common.AuthTypeIDP:
		switch authInfo.AuthProtocol {
		case common.AuthProtocolSAML:
			return saml.AuthenticateAndGetUnscopedToken, nil
		case common.AuthProtocolOIDC:
			return oidc.AuthenticateAndGetUnscopedToken, nil
		default:
			return nil, errors.New(
				"fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". " +
					"Please provide a valid argument and try again")
		}
	case common.AuthTypeIAM:
		iamProvider := func(ctx context.Context, ai common.AuthInfo) (*common.TokenResponse, error) {
			return iam.AuthenticateAndGetUnscopedToken(ai)
		}
		return iamProvider, nil
	default:
		return nil, errors.New(
			"fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". " +
				"Please provide a valid argument and try again")
	}
}

func handleSuccessfulAuthentication(tokenResponse common.TokenResponse, region string) error {
	if tokenResponse.Token.Secret == "" {
		return errors.New("authorization did not succeed. please try again")
	}

	if err := updateOTCInfoFile(tokenResponse, region); err != nil {
		return fmt.Errorf("couldn't update otc info file: %w", err)
	}

	projectsInActiveCloud := iam.GetProjectsInActiveCloud()
	if err := iam.CreateScopedTokenForEveryProject(projectsInActiveCloud.GetProjectNames()); err != nil {
		return fmt.Errorf("couldn't create scoped token for projects: %w", err)
	}

	glog.V(common.InfoLogLevel).Info("info: successfully obtained unscoped token!")
	return nil
}

func AuthenticateAndGetUnscopedToken(loginCtx context.Context, authInfo common.AuthInfo) error {
	if err := config.LoadCloudConfig(authInfo.DomainName); err != nil {
		return fmt.Errorf("couldn't load config: %w", err)
	}

	if config.IsAuthenticationValid() && !authInfo.OverwriteFile {
		glog.V(common.InfoLogLevel).Info(
			"info: will not retrieve unscoped token, because the current one is still valid.\n" +
				"To overwrite the existing unscoped token, pass the \"--overwrite-token\" argument")
		return nil
	}

	glog.V(common.InfoLogLevel).Info("info: retrieving unscoped token for active cloud...")

	provider, err := getTokenProvider(authInfo)
	if err != nil {
		return err
	}

	tokenResponse, err := provider(loginCtx, authInfo)
	if err != nil {
		return fmt.Errorf("couldn't get unscoped token: %w", err)
	}

	return handleSuccessfulAuthentication(*tokenResponse, authInfo.Region)
}

func updateOTCInfoFile(tokenResponse common.TokenResponse, regionCode string) error {
	activeCloud, err := config.GetActiveCloudConfig()
	if err != nil {
		return fmt.Errorf("couldn't get cloud config: %w", err)
	}
	if activeCloud.Domain.Name != tokenResponse.Token.User.Domain.Name {
		// Sanity check: we're in the same cloud as the active cloud
		return errors.New("fatal: authorization made for wrong cloud configuration")
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
	err = config.UpdateCloudConfig(*activeCloud)
	if err != nil {
		return fmt.Errorf("couldn't update config: %w", err)
	}

	return nil
}
