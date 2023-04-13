package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"os"
	"otc-auth/accesstoken"
	"otc-auth/cce"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/openstack"
)

// GoReleaser will set the following 2 ldflags by default
var (
	version = "dev"
	date    = "unknown"
)

const (
	osUsername          = "os-username"
	osPassword          = "os-password"
	overwriteTokenArg   = "overwrite-token"
	osDomainName        = "os-domain-name"
	osUserDomainId      = "os-user-domain-id"
	osProjectName       = "os-project-name"
	totpArg             = "totp"
	idpName             = "idp-name"
	idpUrlArg           = "idp-url"
	clientIdArg         = "client-id"
	clientSecretArg     = "client-secret"
	clusterArg          = "cluster"
	isServiceAccountArg = "service-account"
)

func main() {
	const (
		provideArgumentHelp = "Either provide this argument or set the environment variable"
		overwriteTokenHelp  = "Overrides .otc-info file"
		requiredForIdp      = "Required for authentication with IdP."
	)
	var (
		domainName           *string
		username             *string
		password             *string
		overwriteToken       *bool
		identityProvider     *string
		identityProviderUrl  *string
		isServiceAccount     *bool
		idpCommandHelp       = fmt.Sprintf("The name of the identity provider. Allowed values in the iam section of the OTC UI. %s %s %s", requiredForIdp, provideArgumentHelp, envIdpName)
		idpUrlCommandHelp    = fmt.Sprintf("Url from the identity provider (e.g. ...realms/myrealm/protocol/saml). %s %s %s", requiredForIdp, provideArgumentHelp, envIdpUrl)
		isServiceAccountHelp = "Flag to set if the account is a service account. The service account needs to be configured in your identity provider."
	)

	parser := argparse.NewParser("otc-auth", "OTC-Auth Command Line Interface for managing OTC clouds.")

	// Version
	versionCommand := parser.NewCommand("version", "Returns OTC-Auth's version.")

	// Login & common commands
	loginCommand := parser.NewCommand("login", "Login to the Open Telekom Cloud and receive an unscoped token.")
	username = loginCommand.String("u", osUsername, &argparse.Options{Required: false, Help: fmt.Sprintf("Username for the OTC IAM system. %s %s", provideArgumentHelp, envOsUsername)})
	password = loginCommand.String("p", osPassword, &argparse.Options{Required: false, Help: fmt.Sprintf("Password for the OTC IAM system. %s %s", provideArgumentHelp, envOsPassword)})
	domainName = loginCommand.String("d", osDomainName, &argparse.Options{Required: false, Help: fmt.Sprintf("OTC domain name. %s %s", provideArgumentHelp, envOsDomainName)})
	overwriteToken = loginCommand.Flag("o", overwriteTokenArg, &argparse.Options{Required: false, Help: overwriteTokenHelp, Default: false})
	identityProvider = loginCommand.String("i", idpName, &argparse.Options{Required: false, Help: idpCommandHelp})
	identityProviderUrl = loginCommand.String("", idpUrlArg, &argparse.Options{Required: false, Help: idpUrlCommandHelp})

	// Remove Login information
	removeLoginCommand := loginCommand.NewCommand("remove", "Removes login information for a cloud")

	// Login with IAM
	loginIamCommand := loginCommand.NewCommand("iam", "Login to the Open Telekom Cloud through its Identity and Access Management system.")
	totp := loginIamCommand.String("t", totpArg, &argparse.Options{Required: false, Help: "6-digit time-based one-time password (TOTP) used for the MFA login flow."})
	userDomainId := loginIamCommand.String("", osUserDomainId, &argparse.Options{Required: false, Help: fmt.Sprintf("User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --totp is provided. %s %s", provideArgumentHelp, envOsUserDomainId)})

	// Login with IDP + SAML
	loginIdpSamlCommand := loginCommand.NewCommand("idp-saml", "Login to the Open Telekom Cloud through an Identity Provider and SAML.")

	// Login with IDP + OIDC
	loginIdpOidcCommand := loginCommand.NewCommand("idp-oidc", "Login to the Open Telekom Cloud through an Identity Provider and OIDC.")
	clientId := loginIdpOidcCommand.String("c", clientIdArg, &argparse.Options{Required: false, Help: fmt.Sprintf("Client Id as set on the IdP. %s %s", provideArgumentHelp, envClientId)})
	clientSecret := loginIdpOidcCommand.String("s", clientSecretArg, &argparse.Options{Required: false, Help: fmt.Sprintf("Secret Id as set on the IdP. %s %s", provideArgumentHelp, envClientSecret)})
	isServiceAccount = loginIdpOidcCommand.Flag("", isServiceAccountArg, &argparse.Options{Required: false, Help: isServiceAccountHelp})

	// List Projects
	projectsCommand := parser.NewCommand("projects", "Manage Project Information")
	listProjectsCommand := projectsCommand.NewCommand("list", "List Projects in Active Cloud")

	// Manage Cloud Container Engine
	cceCommand := parser.NewCommand("cce", "Manage Cloud Container Engine.")
	projectName := cceCommand.String("p", osProjectName, &argparse.Options{Required: false, Help: fmt.Sprintf("Name of the project you want to access. %s %s.", provideArgumentHelp, envOsProjectName)})
	cceDomainName := cceCommand.String("d", osDomainName, &argparse.Options{Required: false, Help: fmt.Sprintf("OTC domain name. %s %s", provideArgumentHelp, envOsDomainName)})

	// List clusters
	getClustersCommand := cceCommand.NewCommand("list", "Lists Project Clusters in CCE.")

	// Get Kubernetes Configuration
	getKubeConfigCommand := cceCommand.NewCommand("get-kube-config", "Get remote kube config and merge it with existing local config file.")
	clusterName := getKubeConfigCommand.String("c", clusterArg, &argparse.Options{Required: false, Help: fmt.Sprintf("Name of the clusterArg you want to access %s %s.", provideArgumentHelp, envClusterName)})
	daysValid := getKubeConfigCommand.String("v", "days-valid", &argparse.Options{Required: false, Help: "Period (in days) that the config will be valid", Default: "7"})
	targetLocation := getKubeConfigCommand.String("l", "target-location", &argparse.Options{Required: false, Help: "Where the kube config should be saved, Default: ~/.kube/config"})

	// AK/SK Management
	accessTokenCommand := parser.NewCommand("access-token", "Manage AK/SK.")
	accessTokenCommandCreate := accessTokenCommand.NewCommand("create", "Create new AK/SK.")
	atDomainName := accessTokenCommand.String("d", osDomainName, &argparse.Options{Required: false, Help: fmt.Sprintf("OTC domain name. %s %s", provideArgumentHelp, envOsDomainName)})
	durationSeconds := accessTokenCommandCreate.Int("t", "duration-seconds", &argparse.Options{Required: false, Help: "Lifetime of AK/SK, min 900 seconds.", Default: 900})

	//Openstack Management
	openStackCommand := parser.NewCommand("openstack", "Manage Openstack Integration")
	openStackCommandCreateConfigFile := openStackCommand.NewCommand("config-create", "Creates new clouds.yaml")
	openStackConfigLocation := openStackCommand.String("l", "config-location", &argparse.Options{Required: false, Help: "Where the config should be saved, Default: ~/.config/openstack/clouds.yaml"})

	err := parser.Parse(os.Args)
	if err != nil {
		common.OutputErrorMessageToConsoleAndExit(parser.Usage(err))
	}

	if versionCommand.Happened() {
		_, err := fmt.Fprintf(os.Stdout, "OTC-Auth %s (%s)", version, date)
		if err != nil {
			common.OutputErrorToConsoleAndExit(err, "fatal: could not print tool version.")
		}
	}

	if loginIamCommand.Happened() {
		totpToken, userId := checkMFAFlowIAM(*totp, *userDomainId)
		authInfo := common.AuthInfo{
			AuthType:      authTypeIAM,
			Username:      getUsernameOrThrow(*username),
			Password:      getPasswordOrThrow(*password),
			DomainName:    getDomainNameOrThrow(*domainName),
			Otp:           totpToken,
			UserDomainId:  userId,
			OverwriteFile: *overwriteToken,
		}

		AuthenticateAndGetUnscopedToken(authInfo)
	}

	if loginIdpSamlCommand.Happened() {
		identityProvider, identityProviderUrl := getIdpInfoOrThrow(*identityProvider, *identityProviderUrl)
		authInfo := common.AuthInfo{
			AuthType:      authTypeIDP,
			Username:      getUsernameOrThrow(*username),
			Password:      getPasswordOrThrow(*password),
			DomainName:    getDomainNameOrThrow(*domainName),
			IdpName:       identityProvider,
			IdpUrl:        identityProviderUrl,
			AuthProtocol:  protocolSAML,
			OverwriteFile: *overwriteToken,
		}

		AuthenticateAndGetUnscopedToken(authInfo)
	}

	if loginIdpOidcCommand.Happened() {
		identityProvider, identityProviderUrl := getIdpInfoOrThrow(*identityProvider, *identityProviderUrl)
		authInfo := common.AuthInfo{
			AuthType:         authTypeIDP,
			IdpName:          identityProvider,
			IdpUrl:           identityProviderUrl,
			AuthProtocol:     protocolOIDC,
			DomainName:       getDomainNameOrThrow(*domainName),
			ClientId:         getClientIdOrThrow(*clientId),
			ClientSecret:     findClientSecretOrReturnEmpty(*clientSecret),
			OverwriteFile:    *overwriteToken,
			IsServiceAccount: *isServiceAccount,
		}

		AuthenticateAndGetUnscopedToken(authInfo)
	}

	if removeLoginCommand.Happened() {
		domainNameToRemove := getDomainNameOrThrow(*domainName)
		config.RemoveCloudConfig(domainNameToRemove)
	}

	if listProjectsCommand.Happened() {
		iam.GetProjectsInActiveCloud()
	}

	if cceCommand.Happened() {
		domainName := getDomainNameOrThrow(*cceDomainName)
		config.LoadCloudConfig(domainName)

		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		project := getProjectNameOrThrow(*projectName)

		if getKubeConfigCommand.Happened() {
			cluster := getClusterNameOrThrow(*clusterName)

			kubeConfigParams := cce.KubeConfigParams{
				ProjectName:    project,
				ClusterName:    cluster,
				DaysValid:      *daysValid,
				TargetLocation: *targetLocation,
			}

			cce.GetKubeConfig(kubeConfigParams)
			return
		}

		if getClustersCommand.Happened() {
			cce.GetClusterNames(project)
			return
		}
	}

	if accessTokenCommandCreate.Happened() {
		domainName := getDomainNameOrThrow(*atDomainName)
		config.LoadCloudConfig(domainName)

		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		if *durationSeconds < 900 {
			common.OutputErrorMessageToConsoleAndExit("fatal: argument duration-seconds may not be smaller then 900 seconds")
		}
		accesstoken.CreateAccessToken(*durationSeconds)
	}

	if openStackCommandCreateConfigFile.Happened() {
		openstack.WriteOpenStackCloudsYaml(*openStackConfigLocation)
	}

}
