package saml

import (
	"net/http"

	"otc-auth/common"
)

type CredentialParser interface {
	Parse(resp *http.Response) (*common.TokenResponse, error)
}

type defaultCredentialParser struct{}

func NewDefaultCredentialParser() CredentialParser {
	return &defaultCredentialParser{}
}

func (p *defaultCredentialParser) Parse(resp *http.Response) (*common.TokenResponse, error) {
	return common.GetCloudCredentialsFromResponse(resp)
}
