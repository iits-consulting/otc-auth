package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"otc-auth/common"
)

const (
	envOsUsername        = "OS_USERNAME"
	envOsPassword        = "OS_PASSWORD"
	envOsDomainName      = "OS_DOMAIN_NAME"
	envRegion            = "REGION"
	envOsUserDomainID    = "OS_USER_DOMAIN_ID"
	envOsProjectName     = "OS_PROJECT_NAME"
	envIdpName           = "IDP_NAME"
	envIdpURL            = "IDP_URL"
	envClientID          = "CLIENT_ID"
	envClientSecret      = "CLIENT_SECRET"
	envClusterName       = "CLUSTER_NAME"
	envOidScopes         = "OIDC_SCOPES"
	envOidcScopesDefault = "openid,profile,roles,name,groups,email"

	authTypeIDP = "idp"
	authTypeIAM = "iam"

	protocolSAML = "saml"
	protocolOIDC = "oidc"
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

	return getEnvironmentVariableOrThrow(clusterArg, envClusterName)
}

func getIdpInfoOrThrow(provider string, url string) (string, string) {
	provider = checkIDPProviderIsSet(provider)
	url = checkIdpURLIsSet(url)
	return provider, url
}

func checkIDPProviderIsSet(provider string) string {
	if provider != "" {
		return provider
	}

	return getEnvironmentVariableOrThrow(idpName, envIdpName)
}

func checkIdpURLIsSet(url string) string {
	if url != "" {
		return url
	}

	return getEnvironmentVariableOrThrow(idpURLArg, envIdpURL)
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

func getDurationSecondsOrThrow(durationSeconds int) int {
	if durationSeconds < 900 || durationSeconds > 86400 {
		common.OutputErrorMessageToConsoleAndExit(
			"fatal: token duration must be between 900 and 86400 seconds (15m and 24h).")
	}

	return durationSeconds
}

func getRegionCodeOrThrow(regionCode string) string {
	if regionCode != "" {
		return regionCode
	}

	return getEnvironmentVariableOrThrow(region, envRegion)
}

func checkMFAFlowIAM(otp string, userID string) (string, string) {
	if otp != "" {
		if userID != "" {
			return otp, userID
		}
		userID = getEnvironmentVariableOrThrow(osUserDomainID, envOsUserDomainID)
	}

	return otp, userID
}

func getClientIDOrThrow(id string) string {
	if id != "" {
		return id
	}

	return getEnvironmentVariableOrThrow(clientIDArg, envClientID)
}

func findClientSecretOrReturnEmpty(secret string) string {
	if secret != "" {
		return secret
	} else if secretEnvVar, ok := os.LookupEnv(envClientSecret); ok {
		return secretEnvVar
	} else {
		log.Printf("info: argument --%s not set. Continuing...\n", clientSecretArg)
		return ""
	}
}

func getOidcScopes(scopesFromFlag string) []string {
	if scopesFromFlag != "" {
		return strings.Split(scopesFromFlag, ",")
	}

	scopeFromEnv, ok := os.LookupEnv(envOidScopes)
	if ok {
		return strings.Split(scopeFromEnv, ",")
	}
	return strings.Split(envOidcScopesDefault, ",")
}

func getEnvironmentVariableOrThrow(argument string, envVarName string) string {
	environmentVariable, ok := os.LookupEnv(envVarName)
	if !ok || environmentVariable == "" {
		common.OutputErrorMessageToConsoleAndExit(
			noArgumentProvidedErrorMessage(
				fmt.Sprintf("--%s", argument), envVarName))
	}

	return environmentVariable
}

func noArgumentProvidedErrorMessage(argument string, environmentVariable string) string {
	return fmt.Sprintf(
		"fatal: %s not provided.\n\nPlease make sure the argument %s is provided or the environment variable %s is set.",
		argument, argument, environmentVariable)
}
