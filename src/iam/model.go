package iam

import (
	"encoding/xml"
)

type LoginParams struct {
	AuthType            string
	IdentityProvider    string
	IdentityProviderUrl string
	Username            string
	Password            string
	Protocol            string
	DomainName          string
	Otp                 string
	UserDomainId        string
	ClientId            string
	ClientSecret        string
	OverwriteFile       bool
}

type GetSAMLAssertionResult struct {
	XMLName xml.Name
	Header  struct {
		Response struct {
			ConsumerUrl string `xml:"AssertionConsumerServiceURL,attr"`
		} `xml:"Response"`
	} `xml:"Header"`
}

type GetProjectsResult struct {
	Projects []struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"projects"`
}
type OIDCUsernameAndToken struct {
	BearerToken string
	Claims      struct {
		PreferredUsername string `json:"preferred_username"`
	}
}
