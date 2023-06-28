package login

const (
	envOsUsername        = "OS_USERNAME"
	envOsPassword        = "OS_PASSWORD"
	envOsDomainName      = "OS_DOMAIN_NAME"
	envOsUserDomainId    = "OS_USER_DOMAIN_ID"
	envOsProjectName     = "OS_PROJECT_NAME"
	envIdpName           = "IDP_NAME"
	envIdpUrl            = "IDP_URL"
	envClientId          = "CLIENT_ID"
	envClientSecret      = "CLIENT_SECRET"
	envClusterName       = "CLUSTER_NAME"
	envOidScopes         = "OIDC_SCOPES"
	envOidcScopesDefault = "openid,profile,roles,name,groups,email"

	authTypeIDP = "idp"
	authTypeIAM = "iam"

	protocolSAML = "saml"
	protocolOIDC = "oidc"
)
