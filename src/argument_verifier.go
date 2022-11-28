package main

import (
	"fmt"
	"os"
	"otc-auth/src/common"
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
	envOsProjectName  string = "OS_PROJECT_NAME"
	envClusterName    string = "CLUSTER_NAME"

	authTypeIDP string = "idp"
	authTypeIAM string = "iam"

	protocolSAML string = "saml"
	protocolOIDC string = "oidc"
)

func getProjectNameOrThrow(projectName string) string {
	if projectName != "" {
		return projectName
	}

	return getEnvironmentVariableOrThrow(osProjectName, envOsProjectName)
}

func getClusterNameOrThrow(clusterName string) string {
	if clusterName != "" {
		return clusterName
	}

	return getEnvironmentVariableOrThrow(clusterName, envClusterName)
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

	return getEnvironmentVariableOrThrow(idpName, envIdpName)
}

func checkAuthUrlIsSet(url string) string {
	if url != "" {
		return url
	}

	return getEnvironmentVariableOrThrow(osAuthUrl, envOsAuthUrl)
}

func getUsernameOrThrow(username string) string {
	if username != "" {
		return username
	}

	return getEnvironmentVariableOrThrow(osUsername, envOsUsername)
}

func getPasswordOrThrow(password string) string {
	if password != "" {
		return password
	}

	return getEnvironmentVariableOrThrow(osPassword, envOsPassword)
}

func getDomainNameOrThrow(domainName string) string {
	if domainName != "" {
		return domainName
	}

	return getEnvironmentVariableOrThrow(osDomainName, envOsDomainName)
}

func checkMFAFlowIAM(otp string, userId string) (string, string) {
	if otp != "" {
		if userId != "" {
			return otp, userId
		}
		userId = getEnvironmentVariableOrThrow(osUserDomainId, envOsUserDomainId)
	}

	return otp, userId
}

func getClientIdOrThrow(id string) string {
	if id != "" {
		return id
	}

	return getEnvironmentVariableOrThrow(clientId, envClientId)
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

func getEnvironmentVariableOrThrow(argument string, envVarName string) string {
	environmentVariable, ok := os.LookupEnv(envVarName)
	if !ok || environmentVariable == "" {
		common.OutputErrorMessageToConsoleAndExit(noArgumentProvidedErrorMessage(fmt.Sprintf("--%s", argument), envVarName))
	}

	return environmentVariable
}

func noArgumentProvidedErrorMessage(argument string, environmentVariable string) string {
	return fmt.Sprintf("fatal: %s not provided.\n\nPlease make sure the argument %s is provided or the environment variable %s is set.", argument, argument, environmentVariable)
}
