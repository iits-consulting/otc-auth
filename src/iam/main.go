package iam

import (
	"otc-cli/src/util"
	"time"
)

const IamAuthUrl = "https://iam.eu-de.otc.t-systems.com:443"
const XmlContentType = "text/xml"
const SoapContentType = "application/vnd.paos+xml"
const SoapHeaderInfo = `ver="urn:liberty:paos:2003-08";"urn:oasis:names:tc:SAML:2.0:profiles:SSO:ecp"`

func Login(loginParams LoginParams) {
	if !util.LoginNeeded() {
		println("Previous token still valid. Continue.")
		return
	}
	if loginParams.Protocol != "saml" {
		util.OutputErrorMessageToConsoleAndExit("fatal: invalid protocol.\n\nOnly saml is supported at the moment.")
	}

	println("Retrieving unscoped token...")

	var unscopedToken string
	switch loginParams.AuthType {
	case "idp":
		if loginParams.Protocol == "saml" {
			unscopedToken = getUnscopedSAMLToken(loginParams)
		} else if loginParams.Protocol == "oidc" {
			// login with oidc
			util.OutputErrorMessageToConsoleAndExit("fatal: unsupported login protocol.\n\nCurrently the only supported protocol is \"saml\". Please provide a valid argument and try again.")
		} else {
			util.OutputErrorMessageToConsoleAndExit("fatal: unsupported login protocol.\n\nAllowed values are \"saml\" or \"oidc\". Please provide a valid argument and try again.")
		}
	case "iam":
		unscopedToken = getUserToken(loginParams)
	default:
		util.OutputErrorMessageToConsoleAndExit("fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". Please provide a valid argument and try again.")
	}

	updateOTCInfoFile(loginParams, unscopedToken)
	println("Successfully obtained unscoped token!")
}

func updateOTCInfoFile(loginParams LoginParams, unscopedToken string) {
	otcInformation := util.ReadOrCreateOTCInfoFromFile()

	otcInformation.UnscopedToken.Value = unscopedToken
	valid23Hours := time.Now().Add(time.Hour)
	otcInformation.Username = loginParams.Username
	otcInformation.UnscopedToken.ValidTill = valid23Hours.Format(util.TimeFormat)
	util.UpdateOtcInformation(otcInformation)
}

func GetScopedToken(projectName string) string {
	scopedTokenFormOTCInfoFile := util.GetScopedTokenFromOTCInfo(projectName)
	if scopedTokenFormOTCInfoFile == "" {
		OrderNewScopedToken(projectName)
		return util.GetScopedTokenFromOTCInfo(projectName)
	}
	return scopedTokenFormOTCInfoFile
}

func GetProjectId(projectName string) string {
	return util.FindProjectID(projectName)
}
