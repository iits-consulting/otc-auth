package iam

import (
	util2 "otc-cli/src/util"
	"time"
)

const IamAuthUrl = "https://iam.eu-de.otc.t-systems.com:443"
const XmlContentType = "text/xml"
const SoapContentType = "application/vnd.paos+xml"
const SoapHeaderInfo = `ver="urn:liberty:paos:2003-08";"urn:oasis:names:tc:SAML:2.0:profiles:SSO:ecp"`

func Login(loginParams LoginParams) {
	if !util2.LoginNeeded() {
		println("Previous token still valid. Continue.")
		return
	}
	if loginParams.Protocol != "saml" {
		util2.OutputErrorMessageToConsoleAndExit("fatal: invalid protocol.\n\nOnly saml is supported at the moment.")
	}

	println("Retrieving unscoped token...")

	var unscopedToken string
	switch loginParams.AuthType {
	case "idp":
		unscopedToken = getUnscopedSAMLToken(loginParams)
	case "iam":
		unscopedToken = getUserToken(loginParams)
	default:
		util2.OutputErrorMessageToConsoleAndExit("fatal: unsupported authorization type.\n\nAllowed values are \"idp\" or \"iam\". Please provide a valid argument and try again.")
	}

	updateOTCInfoFile(loginParams, unscopedToken)
	println("Successfully obtained unscoped token!")
}

func updateOTCInfoFile(loginParams LoginParams, unscopedToken string) {
	otcInformation := util2.ReadOrCreateOTCInfoFromFile()

	otcInformation.UnscopedToken.Value = unscopedToken
	valid23Hours := time.Now().Add(time.Hour)
	otcInformation.Username = loginParams.Username
	otcInformation.UnscopedToken.ValidTill = valid23Hours.Format(util2.TimeFormat)
	util2.UpdateOtcInformation(otcInformation)
}

func GetScopedToken(projectName string) string {
	scopedTokenFormOTCInfoFile := util2.GetScopedTokenFromOTCInfo(projectName)
	if scopedTokenFormOTCInfoFile == "" {
		OrderNewScopedToken(projectName)
		return util2.GetScopedTokenFromOTCInfo(projectName)
	}
	return scopedTokenFormOTCInfoFile
}

func GetProjectId(projectName string) string {
	return util2.FindProjectID(projectName)
}
