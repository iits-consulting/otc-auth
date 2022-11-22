package iam

import (
	"fmt"
	"otc-auth/src/util"
	"time"
)

const (
	AuthUrlIam             = "https://iam.eu-de.otc.t-systems.com:443"
	XmlContentType         = "text/xml"
	SoapContentType        = "application/vnd.paos+xml"
	SoapHeaderInfo         = `ver="urn:liberty:paos:2003-08";"urn:oasis:names:tc:SAML:2.0:profiles:SSO:ecp"`
	protocolSAML    string = "saml"
	protocolOIDC    string = "oidc"
)

func Login(loginParams LoginParams) {
	if !util.LoginNeeded(loginParams.OverwriteFile) {
		println("info: will not retrieve unscoped token, because the current one is still valid.\n\nTo overwrite the existing unscoped token, pass the \"--overwrite-token\" argument.")
		return
	}

	println("Retrieving unscoped token...")

	var unscopedToken string
	switch loginParams.AuthType {
	case "idp":
		if loginParams.Protocol == protocolSAML {
			unscopedToken = getUnscopedSAMLToken(loginParams)
		} else if loginParams.Protocol == protocolOIDC {
			unscopedToken, loginParams.Username = getUnscopedOIDCToken(loginParams)
		} else {
			util.OutputErrorMessageToConsoleAndExit("fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". Please provide a valid argument and try again.")
		}
	case "iam":
		unscopedToken = getUserToken(loginParams)
	default:
		util.OutputErrorMessageToConsoleAndExit("fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". Please provide a valid argument and try again.")
	}

	if unscopedToken == "" {
		util.OutputErrorMessageToConsoleAndExit("Authorization did not succeed. Please try again.")
	}
	updateOTCInfoFile(loginParams, unscopedToken)
	println("Successfully obtained unscoped token!")
}

func updateOTCInfoFile(loginParams LoginParams, unscopedToken string) {
	otcInfo := util.ReadOrCreateOTCInfoFromFile()

	otcInfo.UnscopedToken.Value = unscopedToken
	expirationDate := time.Now().Add(time.Hour * 23)
	otcInfo.Username = loginParams.Username
	otcInfo.UnscopedToken.ValidTill = expirationDate.Format(util.TimeFormat)
	println(fmt.Sprintf("Unscoped token valid until %s", expirationDate.Format(util.PrintTimeFormat)))
	util.UpdateOtcInformation(otcInfo)
}

func GetScopedToken(projectName string) string {
	scopedTokenFromOTCInfoFile := util.GetScopedTokenFromOTCInfo(projectName)
	if scopedTokenFromOTCInfoFile == "" {
		GetNewScopedToken(projectName)
		return util.GetScopedTokenFromOTCInfo(projectName)
	}
	return scopedTokenFromOTCInfoFile
}

func GetProjectId(projectName string) string {
	return util.FindProjectID(projectName)
}
