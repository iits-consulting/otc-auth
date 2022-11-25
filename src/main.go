package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"os"
	"otc-auth/src/cce"
	"otc-auth/src/iam"
	"otc-auth/src/util"
	"strings"
)

const (
	osUsername     = "os-username"
	osPassword     = "os-password"
	overwriteToken = "overwrite-token"
	osDomainName   = "os-domain-name"
	osUserDomainId = "os-user-domain-id"
	totp           = "totp"
	idpName        = "idp-name"
	osAuthUrl      = "os-auth-url"
	clientId       = "client-id"
	clientSecret   = "client-secret"
)

func main() {
	parser := argparse.NewParser("otc", "kubectl plugin for OTC")

	//Login
	loginCommand := parser.NewCommand("login", "Login to the Open Telekom Cloud and receive an unscoped token.")

	// Login with IAM
	loginIamCommand := loginCommand.NewCommand("iam", "Login to the Open Telekom Cloud through its Identity and Access Management system.")
	provideArgumentHelp := "Either provide this argument or set it on the environment variable"
	usernameIamLogin := loginIamCommand.String("u", osUsername, &argparse.Options{Required: false, Help: fmt.Sprintf("Username for the OTC IAM system. %s %s", provideArgumentHelp, envOsUsername)})
	passwordIamLogin := loginIamCommand.String("p", osPassword, &argparse.Options{Required: false, Help: fmt.Sprintf("Password for the OTC IAM system. %s %s", provideArgumentHelp, envOsPassword)})
	domainName := loginIamCommand.String("d", osDomainName, &argparse.Options{Required: false, Help: fmt.Sprintf("OTC domain name. %s %s", provideArgumentHelp, envOsDomainName)})
	otp := loginIamCommand.String("t", totp, &argparse.Options{Required: false, Help: "6-digit time-based one-time password (TOTP) used for the MFA login flow."})
	userDomainId := loginIamCommand.String("i", osUserDomainId, &argparse.Options{Required: false, Help: fmt.Sprintf("User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --otp is provided. %s %s", provideArgumentHelp, envOsUserDomainId)})
	overwriteTokenHelp := "Overrides .otc-info file."
	overwriteTokenLoginIam := loginIamCommand.Flag("o", overwriteToken, &argparse.Options{Required: false, Help: overwriteTokenHelp, Default: false})

	// Login with IDP + SAML
	loginIdpSamlCommand := loginCommand.NewCommand("idp-saml", "Login to the Open Telekom Cloud through an Identity Provider and SAML.")
	usernameSamlLogin := loginIdpSamlCommand.String("u", osUsername, &argparse.Options{Required: false, Help: fmt.Sprintf("Username for the IdP. %s %s", provideArgumentHelp, envOsUsername)})
	passwordSamlLogin := loginIdpSamlCommand.String("p", osPassword, &argparse.Options{Required: false, Help: fmt.Sprintf("Password for the IdP. %s %s", provideArgumentHelp, envOsPassword)})
	requiredForIdp := "Required for authentication with IdP."
	idpCommandHelp := fmt.Sprintf("The name of the identity provider. Allowed values in the iam section of the OTC UI. %s %s %s", requiredForIdp, provideArgumentHelp, envIdpName)
	identityProviderSamlLogin := loginIdpSamlCommand.String("i", idpName, &argparse.Options{Required: false, Help: idpCommandHelp})
	idpUrlCommandHelp := fmt.Sprintf("Url from the identity provider (e.g. ...realms/myrealm/protocol/saml). %s %s %s", requiredForIdp, provideArgumentHelp, envOsAuthUrl)
	overwriteTokenLoginSaml := loginIdpSamlCommand.Flag("o", overwriteToken, &argparse.Options{Required: false, Help: overwriteTokenHelp, Default: false})
	identityProviderUrlSamlLogin := loginIdpSamlCommand.String("", osAuthUrl, &argparse.Options{Required: false, Help: idpUrlCommandHelp})

	// Login with IDP + OIDC
	loginIdpOidcCommand := loginCommand.NewCommand("idp-oidc", "Login to the Open Telekom Cloud through an Identity Provider and OIDC.")
	identityProviderOidcLogin := loginIdpOidcCommand.String("i", idpName, &argparse.Options{Required: false, Help: idpCommandHelp})
	clientIdCommand := loginIdpOidcCommand.String("c", clientId, &argparse.Options{Required: false, Help: fmt.Sprintf("Client Id as set on the IdP. %s %s", provideArgumentHelp, envClientId)})
	clientSecretCommand := loginIdpOidcCommand.String("s", clientSecret, &argparse.Options{Required: false, Help: fmt.Sprintf("Secret Id as set on the IdP. %s %s", provideArgumentHelp, envClientSecret)})
	overwriteTokenLoginOidc := loginIdpOidcCommand.Flag("o", overwriteToken, &argparse.Options{Required: false, Help: overwriteTokenHelp, Default: false})
	identityProviderUrlOidcLogin := loginIdpOidcCommand.String("", osAuthUrl, &argparse.Options{Required: false, Help: idpUrlCommandHelp})

	// Manage Cloud Container Engine
	cceCommand := parser.NewCommand("cce", "Manage Cloud Container Engine")
	projectName := cceCommand.String("p", "os-project-name", &argparse.Options{Required: true, Help: "Name of the project you want to access"})

	// List clusters
	getClustersCommand := cceCommand.NewCommand("list", "List Cluster Names")

	// Get Kubernetes Configuration
	getKubeConfigCommand := cceCommand.NewCommand("get-kube-config", "Get remote kube config and merge it with existing local config file")
	clusterName := getKubeConfigCommand.String("c", "cluster", &argparse.Options{Required: true, Help: "Name of the cluster you want to access"})
	daysValid := getKubeConfigCommand.String("v", "days-valid", &argparse.Options{Required: false, Help: "Period (in days) that the config will be valid", Default: "7"})

	// AK/SK Management
	accessTokenCommand := parser.NewCommand("access-token", "Manage AK/SK")
	accessTokenCommandCreate := accessTokenCommand.NewCommand("create", "Create new AK/SK")
	durationSeconds := accessTokenCommandCreate.Int("d", "duration-seconds", &argparse.Options{Required: false, Help: "Lifetime of AK/SK, min 900 seconds", Default: 900})

	err := parser.Parse(os.Args)
	if err != nil {
		util.OutputErrorMessageToConsoleAndExit(parser.Usage(err))
	}

	if loginIamCommand.Happened() {
		loginParams := iam.LoginParams{
			AuthType:      authTypeIAM,
			Username:      *usernameIamLogin,
			Password:      *passwordIamLogin,
			DomainName:    *domainName,
			Otp:           *otp,
			UserDomainId:  *userDomainId,
			OverwriteFile: *overwriteTokenLoginIam,
		}

		loginParams = CheckLoginParamsOrThrow(authTypeIAM, loginParams)
		iam.Login(loginParams)
	}

	if loginIdpSamlCommand.Happened() {
		loginParams := iam.LoginParams{
			AuthType:            authTypeIDP,
			Username:            *usernameSamlLogin,
			Password:            *passwordSamlLogin,
			IdentityProvider:    *identityProviderSamlLogin,
			IdentityProviderUrl: *identityProviderUrlSamlLogin,
			Protocol:            protocolSAML,
			OverwriteFile:       *overwriteTokenLoginSaml,
		}

		loginParams = CheckLoginParamsOrThrow(authTypeIDP, loginParams)
		iam.Login(loginParams)
	}

	if loginIdpOidcCommand.Happened() {
		loginParams := iam.LoginParams{
			AuthType:            authTypeIDP,
			IdentityProvider:    *identityProviderOidcLogin,
			IdentityProviderUrl: *identityProviderUrlOidcLogin,
			Protocol:            protocolOIDC,
			ClientId:            *clientIdCommand,
			ClientSecret:        *clientSecretCommand,
			OverwriteFile:       *overwriteTokenLoginOidc,
		}

		loginParams = CheckLoginParamsOrThrow(authTypeIDP, loginParams)
		iam.Login(loginParams)
	}

	if !loginIamCommand.Happened() && !loginIdpSamlCommand.Happened() && !loginIdpOidcCommand.Happened() {
		if util.LoginNeeded(false) {
			util.OutputErrorMessageToConsoleAndExit("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}
	}

	if cceCommand.Happened() {
		iam.GetScopedToken(*projectName)
		if getKubeConfigCommand.Happened() {
			kubeConfigParams := cce.KubeConfigParams{
				ProjectName: *projectName,
				ClusterName: *clusterName,
				DaysValid:   *daysValid,
			}
			newKubeConfigData := cce.GetKubeConfig(kubeConfigParams)
			cce.MergeKubeConfig(*projectName, *clusterName, newKubeConfigData)
			println(fmt.Sprintf("Successfully fetched and merge kube config for cce cluster %s.", kubeConfigParams.ClusterName))
			return
		}
		if getClustersCommand.Happened() {
			projectName := *projectName
			println(fmt.Sprintf("CCE Clusters inside the project %s:\n%s", projectName, strings.Join(cce.GetClusterNames(projectName), ",\n")))
		}

	}

	if accessTokenCommandCreate.Happened() {
		if *durationSeconds < 900 {
			util.OutputErrorMessageToConsoleAndExit("fatal: argument duration-seconds may not be smaller then 900 seconds")
		}
		AccessTokeCreateParams := iam.AccessTokenCreateParams{
			DurationSeconds: *durationSeconds,
		}
		iam.CreateAccessToken(AccessTokeCreateParams)
	}
}
