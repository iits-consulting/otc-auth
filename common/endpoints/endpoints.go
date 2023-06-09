package endpoints

import (
	"errors"
	"fmt"

	"otc-auth/common"
)

const (
	protocols = "protocols"
	auth      = "auth"
)

func BaseURLIam(region string) string {
	if region == "" {
		common.OutputErrorToConsoleAndExit(errors.New("empty region supplied, can't generate IAM URL"))
	}
	return fmt.Sprintf("https://iam.%s.otc.t-systems.com:443", region)
}

func IdentityProviders(identityProvider string, protocol string, region string) string {
	identityProviders := fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers", BaseURLIam(region))
	return fmt.Sprintf("%s/%s/%s/%s/%s", identityProviders, identityProvider, protocols, protocol, auth)
}
