package main

import (
	"fmt"
	"otc-auth/src/common"
	"otc-auth/src/iam"
	"otc-auth/src/oidc"
	"otc-auth/src/saml"
	"time"
)

func AuthenticateAndGetUnscopedToken(authInfo common.AuthInfo) {
	if !common.IsAuthenticationValid(authInfo.OverwriteFile) {
		println("info: will not retrieve unscoped token, because the current one is still valid.\n\nTo overwrite the existing unscoped token, pass the \"--overwrite-token\" argument.")
		return
	}

	println("Retrieving unscoped token...")

	var unscopedToken string
	switch authInfo.AuthType {
	case "idp":
		if authInfo.Protocol == protocolSAML {
			unscopedToken = saml.AuthenticateAndGetUnscopedToken(authInfo)
		} else if authInfo.Protocol == protocolOIDC {
			unscopedToken, authInfo.Username = oidc.AuthenticateAndGetUnscopedToken(authInfo)
		} else {
			common.OutputErrorMessageToConsoleAndExit("fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". Please provide a valid argument and try again.")
		}
	case "iam":
		unscopedToken = iam.AuthenticateAndGetUnscopedToken(authInfo)
	default:
		common.OutputErrorMessageToConsoleAndExit("fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". Please provide a valid argument and try again.")
	}

	if unscopedToken == "" {
		common.OutputErrorMessageToConsoleAndExit("Authorization did not succeed. Please try again.")
	}
	updateOTCInfoFile(authInfo, unscopedToken)
	println("Successfully obtained unscoped token!")
}

func updateOTCInfoFile(authInfo common.AuthInfo, unscopedToken string) {
	otcInfo := common.ReadOrCreateOTCAuthCredentialsFile()

	otcInfo.UnscopedToken.Value = unscopedToken
	expirationDate := time.Now().Add(time.Hour * 23)
	otcInfo.Username = authInfo.Username
	otcInfo.UnscopedToken.ValidTill = expirationDate.Format(common.TimeFormat)
	println(fmt.Sprintf("Unscoped token valid until %s", expirationDate.Format(common.PrintTimeFormat)))
	common.UpdateOtcInformation(otcInfo)
}
