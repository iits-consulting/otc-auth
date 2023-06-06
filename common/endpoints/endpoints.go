package endpoints

import (
	"fmt"
)

const (
	BaseURLIam = "https://iam.eu-de.otc.t-systems.com:443"
	protocols  = "protocols"
	auth       = "auth"
)

func IdentityProviders(identityProvider string, protocol string) string {
	identityProviders := fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers", BaseURLIam)
	return fmt.Sprintf("%s/%s/%s/%s/%s", identityProviders, identityProvider, protocols, protocol, auth)
}
