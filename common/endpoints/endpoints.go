package endpoints

import (
	"fmt"
)

const (
	BaseUrlIam = "https://iam.eu-de.otc.t-systems.com:443"
	protocols  = "protocols"
	auth       = "auth"
)

var identityProviders = fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers", BaseUrlIam)

func IdentityProviders(identityProvider string, protocol string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", identityProviders, identityProvider, protocols, protocol, auth)
}
