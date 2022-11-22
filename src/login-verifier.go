package main

import (
	"fmt"
	"os"
	"otc-auth/src/iam"
	"otc-auth/src/util"
)

const (
	envOsAuthUrl      string = "OS_AUTH_URL"
	envOsUsername     string = "OS_USERNAME"
	envOsPassword     string = "OS_PASSWORD"
	envOsDomainName   string = "OS_DOMAIN_NAME"
	envOsUserDomainId string = "OS_USER_DOMAIN_ID"
	envIdpName        string = "IDP_NAME"
	envClientId       string = "CLIENT_ID"
	envClientSecret   string = "CLIENT_SECRET"

	authTypeIDP string = "idp"
	authTypeIAM string = "iam"

	protocolSAML string = "saml"
	protocolOIDC string = "oidc"
)

func CheckLoginParamsOrThrow(authType string, params iam.LoginParams) (loginParams iam.LoginParams) {
	loginParams.OverwriteFile = params.OverwriteFile
	switch authType {
	case authTypeIAM:
		loginParams.AuthType = authTypeIAM
		loginParams.Username = getUsernameOrThrow(params.Username)
		loginParams.Password = getPasswordOrThrow(params.Password)
		loginParams.DomainName = getDomainNameOrThrow(params.DomainName)
		loginParams.Otp, loginParams.UserDomainId = checkMFAFlowIAM(params.Otp, params.UserDomainId)
	case authTypeIDP:
		switch params.Protocol {
		case protocolSAML:
			loginParams.AuthType = authTypeIDP
			loginParams.Protocol = protocolSAML
			loginParams.Username = getUsernameOrThrow(params.Username)
			loginParams.Password = getPasswordOrThrow(params.Password)
			loginParams.IdentityProvider, loginParams.IdentityProviderUrl = getIdpInfoOrThrow(params.IdentityProvider, params.IdentityProviderUrl)

		case protocolOIDC:
			loginParams.AuthType = authTypeIDP
			loginParams.Protocol = protocolOIDC
			loginParams.IdentityProvider, loginParams.IdentityProviderUrl = getIdpInfoOrThrow(params.IdentityProvider, params.IdentityProviderUrl)
			loginParams.ClientId = getClientIdOrThrow(params.ClientId)
			loginParams.ClientSecret = findClientSecretOrReturnEmpty(params.ClientSecret)
		default:
			util.OutputErrorMessageToConsoleAndExit("fatal: incorrect login command.\n\nPossible login commands are \"login iam\", \"login idp-saml\", and \"login idp-oidc\".")
		}
	default:
		util.OutputErrorMessageToConsoleAndExit("fatal: incorrect login command.\n\nPossible login commands are \"login iam\", \"login idp-saml\", and \"login idp-oidc\".")
	}

	return
}

func getIdpInfoOrThrow(provider string, url string) (string, string) {
	provider = checkIDPProviderIsSet(provider)
	url = checkAuthUrlIsSet(url)
	return provider, url
}

func checkIDPProviderIsSet(provider string) string {
	if provider != "" {
		return provider
	}

	idpProviderNameEnvVar, ok := os.LookupEnv(envIdpName)
	if !ok || idpProviderNameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", idpName), envIdpName))
	}

	return idpProviderNameEnvVar
}

func checkAuthUrlIsSet(url string) string {
	if url != "" {
		return url
	}

	idpUrlEnvVar, ok := os.LookupEnv(envOsAuthUrl)
	if !ok || idpUrlEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", osAuthUrl), envOsAuthUrl))
	}

	return idpUrlEnvVar
}

func getUsernameOrThrow(username string) string {
	if username != "" {
		return username
	}

	usernameEnvVar, ok := os.LookupEnv(envOsUsername)
	if !ok || usernameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", osUsername), envOsUsername))
	}

	return usernameEnvVar
}

func getPasswordOrThrow(password string) string {
	if password != "" {
		return password
	}

	passwordEnvVar, ok := os.LookupEnv(envOsPassword)
	if !ok || passwordEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", osPassword), envOsPassword))
	}

	return passwordEnvVar
}

func getDomainNameOrThrow(domainName string) string {
	if domainName != "" {
		return domainName
	}

	domainNameEnvVar, ok := os.LookupEnv(envOsDomainName)
	if !ok || domainNameEnvVar == "" {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", osDomainName), envOsDomainName))
	}

	return domainNameEnvVar
}

func checkMFAFlowIAM(otp string, userId string) (string, string) {
	if otp != "" {
		if userId != "" {
			return otp, userId
		}

		userIdEnvVar, ok := os.LookupEnv(envOsUserDomainId)
		if !ok {
			util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", osUserDomainId), envOsUserDomainId))
		}

		userId = userIdEnvVar
	}

	return otp, userId
}

func getClientIdOrThrow(id string) string {
	if id != "" {
		return id
	}

	idEnvVar, ok := os.LookupEnv(envClientId)
	if !ok {
		util.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", clientId), envClientId))
	}

	return idEnvVar
}

func findClientSecretOrReturnEmpty(secret string) string {
	if secret != "" {
		return secret
	} else if secretEnvVar, ok := os.LookupEnv(envClientSecret); ok {
		return secretEnvVar
	} else {
		println(fmt.Sprintf("info: argument --%s not set. Continuing...\n", clientSecret))
		return ""
	}
}

func noArgumentProvidedErrorMessage(argument string, environmentVariable string) string {
	return fmt.Sprintf("fatal: %s not provided.\n\nPlease make sure the argument %s is provided or the environment variable %s is set.", argument, argument, environmentVariable)
}
