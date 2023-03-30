package endpoints

import (
	"fmt"
)

const (
	baseUrlIam = "https://iam.eu-de.otc.t-systems.com:443"
	baseUrlCce = "https://cce.eu-de.otc.t-systems.com:443"
	protocols  = "protocols"
	auth       = "auth"
)

var (
	IamProjects       = fmt.Sprintf("%s/v3/projects", baseUrlIam)
	IamTokens         = fmt.Sprintf("%s/v3/auth/tokens", baseUrlIam)
	IamSecurityTokens = fmt.Sprintf("%s/v3.0/OS-CREDENTIAL/securitytokens", baseUrlIam)
	identityProviders = fmt.Sprintf("%s/v3/OS-FEDERATION/identity_providers", baseUrlIam)

	cceProjects = fmt.Sprintf("%s/api/v3/projects", baseUrlCce)
	clusters    = "clusters"
	clusterCert = "clustercert"
)

func IdentityProviders(identityProvider string, protocol string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", identityProviders, identityProvider, protocols, protocol, auth)
}

func Clusters(projectId string) string {
	return fmt.Sprintf("%s/%s/%s", cceProjects, projectId, clusters)
}

func ClusterCert(projectId string, clusterId string) string {
	clusters := Clusters(projectId)
	return fmt.Sprintf("%s/%s/%s", clusters, clusterId, clusterCert)
}
