package main

import (
	"fmt"
	"github.com/akamensky/argparse"
	"os"
	"otc-cli/cce"
	"otc-cli/iam"
	"otc-cli/util"
	"strings"
)

const osAuthTypeEnv string = "OS_AUTH_TYPE"
const osAuthIDPNameEnv string = "OS_AUTH_IDP_NAME"
const osIDPUrlEnv string = "OS_IDP_URL"
const osUsernameEnv string = "OS_USERNAME"
const osPasswordEnv string = "OS_PASSWORD"
const osDomainNameEnv string = "OS_DOMAIN_NAME"
const osUserIdEnv string = "OS_USER_ID"
const authTypeIDP string = "idp"
const authTypeIAM string = "iam"

func main() {
	parser := argparse.NewParser("otc", "kubectl plugin for OTC")

	//Login
	loginCommand := parser.NewCommand("login", "Use credentials to login and generate an unscoped token")

	provideArgumentHelp := "Either provide this argument or set it on the environment variable"
	authTypeCommandHelp := fmt.Sprintf("Allowed values are idp or iam (from OTC). %s %s", provideArgumentHelp, osAuthTypeEnv)
	authType := loginCommand.String("a", "os-auth-type", &argparse.Options{Required: false, Help: authTypeCommandHelp})
	requiredIfAuthTypeIsIDP := "Required if --os-auth-type is set to idp."
	idpCommandHelp := fmt.Sprintf("The name of the identity provider. Allowed values in the iam section of the OTC UI. %s %s %s", requiredIfAuthTypeIsIDP, provideArgumentHelp, osAuthIDPNameEnv)
	identityProvider := loginCommand.String("I", "os-auth-idp-name", &argparse.Options{Required: false, Help: idpCommandHelp})
	idpUrlCommandHelp := fmt.Sprintf("Url from the identity provider (e.g. ...realms/myrealm/protocol/saml). %s %s %s", requiredIfAuthTypeIsIDP, provideArgumentHelp, osIDPUrlEnv)
	identityProviderUrl := loginCommand.String("i", "os-idp-url", &argparse.Options{Required: false, Help: idpUrlCommandHelp})
	username := loginCommand.String("U", "os-username", &argparse.Options{Required: false, Help: fmt.Sprintf("Username either from idp or OTC iam. %s %s", provideArgumentHelp, osUsernameEnv)})
	password := loginCommand.String("P", "os-password", &argparse.Options{Required: false, Help: fmt.Sprintf("Password either from idp or OTC iam. %s %s", provideArgumentHelp, osPasswordEnv)})
	protocol := loginCommand.String("p", "os-protocol", &argparse.Options{Required: false, Help: "Accepted values are oidc or saml, currently only saml is supported.", Default: "saml"})
	domainName := loginCommand.String("d", "os-domain-name", &argparse.Options{Required: false, Help: "OTC domain name. Required if --os-auth-type is set to iam."})
	otp := loginCommand.String("o", "otp", &argparse.Options{Required: false, Help: "Token used for MFA. Currently only supported for the iam login flow."})
	userId := loginCommand.String("u", "os-user-id", &argparse.Options{Required: false, Help: fmt.Sprintf("User Id used for MFA, can be obtained on the \"My Credentials page\" on the otc. Required if --os-auth-type is set to iam and --otp is provided. %s %s", provideArgumentHelp, osUserIdEnv)})

	//Get Kubernetes Config
	cceCommand := parser.NewCommand("cce", "Manage CCE")
	projectName := cceCommand.String("p", "project", &argparse.Options{Required: true, Help: "Name of the project you want to access"})

	getClustersCommand := cceCommand.NewCommand("list", "List Cluster Names")

	getKubeConfigCommand := cceCommand.NewCommand("get-kube-config", "Get remote kube config and merge it with existing local config file")
	clusterName := getKubeConfigCommand.String("c", "cluster", &argparse.Options{Required: true, Help: "Name of the cluster you want to access"})
	daysValid := getKubeConfigCommand.String("v", "days-valid", &argparse.Options{Required: false, Help: "Period (in days) that the config will be valid", Default: "7"})

	//AK/SK Management
	accessTokenCommand := parser.NewCommand("access-token", "Manage AK/SK")
	accessTokenCommandCreate := accessTokenCommand.NewCommand("create", "Create new AK/SK")
	durationSeconds := accessTokenCommandCreate.Int("s", "duration-seconds", &argparse.Options{Required: false, Help: "Lifetime of AK/SK, min 900 seconds", Default: 900})

	err := parser.Parse(os.Args)
	if err != nil {
		util.OutputErrorMessageToConsoleAndExit(parser.Usage(err))
	}

	if loginCommand.Happened() {
		authType = getAuthTypeOrThrow(authType)
		identityProvider, identityProviderUrl = getIDPInfoOrThrow(authType, identityProvider, identityProviderUrl)
		username = getUsernameOrThrow(username)
		password = getPasswordOrThrow(password)
		domainName = getDomainNameOrThrow(authType, domainName)
		authType, otp, userId = checkMFAFlowIAM(authType, otp, userId)

		loginParams := iam.LoginParams{
			AuthType:            *authType,
			IdentityProvider:    *identityProvider,
			IdentityProviderUrl: *identityProviderUrl,
			Username:            *username,
			Password:            *password,
			Protocol:            *protocol,
			DomainName:          *domainName,
			Otp:                 *otp,
			UserId:              *userId,
		}
		iam.Login(loginParams)
		return
	}
	if util.LoginNeeded() {
		util.OutputErrorMessageToConsoleAndExit("fatal: no unscoped token found.\n\nPlease run the \"otc login\" command first")
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
			println("Successfully fetched and merge kube config for cce cluster " + kubeConfigParams.ClusterName)
			return
		}
		if getClustersCommand.Happened() {
			println("CCE Clusters inside the project " + *projectName + ": " + strings.Join(cce.GetClusterNames(*projectName), ","))
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

func getAuthTypeOrThrow(authType *string) *string {
	if authType != nil && *authType != "" {
		return authType
	}

	authTypeEnvVar, ok := os.LookupEnv(osAuthTypeEnv)
	if !ok || authTypeEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--os-auth-type", osAuthTypeEnv))
	}
	authType = &authTypeEnvVar
	return authType
}

func getIDPInfoOrThrow(authType *string, provider *string, url *string) (*string, *string) {
	if *authType != authTypeIDP {
		return provider, url
	}

	provider = checkIDPProviderIsSet(provider)
	url = checkIDPUrlIsSet(url)
	return provider, url
}

func checkIDPProviderIsSet(provider *string) *string {
	if provider != nil && *provider != "" {
		return provider
	}

	idpProviderNameEnvVar, ok := os.LookupEnv(osAuthIDPNameEnv)
	if !ok || idpProviderNameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--os-auth-idp-name", osAuthIDPNameEnv))
	}

	provider = &idpProviderNameEnvVar
	return provider
}

func checkIDPUrlIsSet(url *string) *string {
	if url != nil && *url != "" {
		return url
	}

	idpUrlEnvVar, ok := os.LookupEnv(osIDPUrlEnv)
	if !ok || idpUrlEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--os-idp-url", osIDPUrlEnv))
	}

	url = &idpUrlEnvVar
	return url
}

func getUsernameOrThrow(username *string) *string {
	if username != nil && *username != "" {
		return username
	}

	usernameEnvVar, ok := os.LookupEnv(osUsernameEnv)
	if !ok || usernameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--username", osUsernameEnv))
	}
	username = &usernameEnvVar
	return username
}

func getPasswordOrThrow(password *string) *string {
	if password != nil && *password != "" {
		return password
	}

	passwordEnvVar, ok := os.LookupEnv(osPasswordEnv)
	if !ok || passwordEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--password", osPasswordEnv))
	}
	password = &passwordEnvVar
	return password
}

func getDomainNameOrThrow(authType *string, domainName *string) *string {
	if authType != nil && *authType != authTypeIAM {
		return domainName
	}

	if domainName != nil && *domainName != "" {
		return domainName
	}

	domainNameEnvVar, ok := os.LookupEnv(osDomainNameEnv)
	if !ok || domainNameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--os-domain-name", osDomainNameEnv))
	}
	domainName = &domainNameEnvVar
	return domainName
}

func checkMFAFlowIAM(authType *string, otp *string, userId *string) (*string, *string, *string) {
	if authType != nil && *authType != authTypeIAM {
		return authType, otp, userId
	}

	if otp != nil && *otp != "" {
		if userId != nil && *userId != "" {
			return authType, otp, userId
		}

		userIdEnvVar, ok := os.LookupEnv(osUserIdEnv)
		if !ok {
			util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage("--os-user-id", osUserIdEnv))
		}

		userId = &userIdEnvVar
	}

	return authType, otp, userId
}

func noArgumentProvidedErrorMessage(argument string, environmentVariable string) string {
	return fmt.Sprintf("fatal: %s not provided.\n\nPlease make sure the argument %s is provided or the environment variable %s is set.", argument, argument, environmentVariable)
}
