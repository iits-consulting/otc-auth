/*
Copyright © 2023 IITS-Consulting

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package cmd

import (
	"fmt"
	"github.com/spf13/pflag"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/login"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: loginCmdHelp,
}

var loginIamCmd = &cobra.Command{
	Use:     "iam",
	Short:   loginIamCmdHelp,
	Long:    loginIamCmdLongHelp,
	Example: loginIamCmdExample,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd, loginIamFlagToEnv)
	},
	Run: func(cmd *cobra.Command, args []string) {
		authInfo := common.AuthInfo{
			AuthType:      "iam",
			Username:      username,
			Password:      password,
			DomainName:    domainName,
			Otp:           totp,
			UserDomainID:  userDomainId,
			OverwriteFile: overwriteToken,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginIdpSamlCmd = &cobra.Command{
	Use:     "idp-saml",
	Short:   loginIdpSamlCmdHelp,
	Long:    "Here we can put a longer description of this command", // TODO
	Example: "Here comes an example usage of this command",          // TODO
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd, loginIdpSamlOidcFlagToEnv)
	},
	Run: func(cmd *cobra.Command, args []string) {
		authInfo := common.AuthInfo{
			AuthType:      "idp",
			Username:      username,
			Password:      password,
			DomainName:    domainName,
			IdpName:       idpName,
			IdpURL:        idpUrl,
			AuthProtocol:  "saml",
			OverwriteFile: overwriteToken,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginIdpOidcCmd = &cobra.Command{
	Use:     "idp-oidc",
	Short:   loginIdpOidcCmdHelp,                                    // TODO
	Long:    "Here we can put a longer description of this command", // TODO
	Example: "Here comes an example usage of this command",          // TODO
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd, loginIdpSamlOidcFlagToEnv)
	},
	Run: func(cmd *cobra.Command, args []string) {
		authInfo := common.AuthInfo{
			AuthType:      "idp",
			Username:      username,
			Password:      password,
			DomainName:    domainName,
			IdpName:       idpName,
			IdpURL:        idpUrl,
			AuthProtocol:  "oidc",
			OverwriteFile: overwriteToken,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   loginRemoveCmdHelp,
	Long:    "Here we can put a longer description of this command", // TODO
	Example: "Here comes an example usage of this command",          // TODO
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd, loginRemoveFlagToEnv)
	},
	Run: func(cmd *cobra.Command, args []string) {
		config.RemoveCloudConfig(domainName)
	},
}

var (
	username       string
	password       string
	domainName     string
	overwriteToken bool
	idpName        string
	idpUrl         string
	totp           string
	userDomainId   string

	loginIamFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIdFlag: userDomainIdEnv,
		idpNameFlag:      idpNameEnv,
		idpUrlFlag:       idpUrlEnv,
	}

	loginIdpSamlOidcFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIdFlag: userDomainIdEnv,
		idpNameFlag:      idpNameEnv,
		idpUrlFlag:       idpUrlEnv,
	}

	loginRemoveFlagToEnv = map[string]string{
		userDomainIdFlag: userDomainIdEnv,
	}
)

/*
initializeConfig is a helper function which sets the environment variable for a flag. It gives precedence to the flag,
meaning that the env is only taken if the flag is empty. It assigns the environment variables to the flags which are
defined in the map flagToEnvMap.
*/
func initializeConfig(cmd *cobra.Command, flagToEnvMapping map[string]string) error {
	v := viper.New()
	v.AutomaticEnv()

	cmd.Flags().VisitAll(
		func(f *pflag.Flag) {
			configName, ok := flagToEnvMapping[f.Name]
			if !ok {
				return
			}
			if !f.Changed && v.IsSet(configName) {
				val := v.Get(configName)
				_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}
		})
	return nil
}

func init() {
	RootCmd.AddCommand(loginCmd)

	loginCmd.AddCommand(loginIamCmd)
	loginIamCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIamCmd.MarkFlagRequired(usernameFlag)
	loginIamCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIamCmd.MarkFlagRequired(passwordFlag)
	loginIamCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIamCmd.MarkFlagRequired(domainNameFlag)
	loginIamCmd.Flags().BoolVarP(&overwriteToken, overwriteTokenFlag, overwriteTokenShortFlag, false, overwriteTokenUsage)
	loginIamCmd.Flags().StringVarP(&totp, totpFlag, totpShortFlag, "", totpUsage)
	loginIamCmd.Flags().StringVarP(&userDomainId, userDomainIdFlag, "", "", userDomainIdUsage)
	loginIamCmd.MarkFlagsRequiredTogether(totpFlag, userDomainIdFlag)

	loginCmd.AddCommand(loginIdpSamlCmd)
	loginIdpSamlCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIdpSamlCmd.MarkFlagRequired(usernameFlag)
	loginIdpSamlCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIdpSamlCmd.MarkFlagRequired(passwordFlag)
	loginIdpSamlCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpSamlCmd.MarkFlagRequired(domainNameFlag)
	loginIdpSamlCmd.Flags().BoolVarP(&overwriteToken, overwriteTokenFlag, overwriteTokenShortFlag, false, overwriteTokenUsage)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpSamlCmd.MarkPersistentFlagRequired(idpNameFlag)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpUrl, idpUrlFlag, "", "", idpUrlUsage)
	loginIdpSamlCmd.MarkPersistentFlagRequired(idpUrlFlag)

	loginCmd.AddCommand(loginIdpOidcCmd)
	loginIdpOidcCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIdpOidcCmd.MarkFlagRequired(usernameFlag)
	loginIdpOidcCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIdpOidcCmd.MarkFlagRequired(passwordFlag)
	loginIdpOidcCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpOidcCmd.MarkFlagRequired(domainNameFlag)
	loginIdpOidcCmd.Flags().BoolVarP(&overwriteToken, overwriteTokenFlag, overwriteTokenShortFlag, false, overwriteTokenUsage)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpOidcCmd.MarkPersistentFlagRequired(idpNameFlag)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpUrl, idpUrlFlag, "", "", idpUrlUsage)
	loginIdpOidcCmd.MarkPersistentFlagRequired(idpUrlFlag)

	loginCmd.AddCommand(loginRemoveCmd)
	loginRemoveCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginRemoveCmd.MarkFlagRequired(domainNameFlag)

}

const (
	loginCmdHelp            = "Login to the Open Telekom Cloud and receive an unscoped token."
	loginIamCmdHelp         = "Login to the Open Telekom Cloud through its Identity and Access Management system."
	loginIamCmdLongHelp     = "Login to the Open Telekom Cloud and receive an unscoped token."
	loginIamCmdExample      = "otc-auth login iam --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName"
	loginIdpSamlCmdHelp     = "Login to the Open Telekom Cloud through an Identity Provider and SAML."
	loginIdpOidcCmdHelp     = "Login to the Open Telekom Cloud through an Identity Provider and OIDC."
	loginRemoveCmdHelp      = "Removes login information for a cloud"
	usernameFlag            = "os-username"
	usernameShortFlag       = "u"
	usernameEnv             = "OS_USERNAME"
	usernameUsage           = "Username for the OTC IAM system. Either provide this argument or set the environment variable " + usernameEnv + "."
	passwordFlag            = "os-password"
	passwordShortFlag       = "p"
	passwordEnv             = "OS_PASSWORD"
	passwordUsage           = "Password for the OTC IAM system. Either provide this argument or set the environment variable " + passwordEnv + "."
	domainNameFlag          = "os-domain-name"
	domainNameShortFlag     = "d"
	domainNameEnv           = "OS_DOMAIN_NAME"
	domainNameUsage         = "OTC domain name. Either provide this argument or set the environment variable " + domainNameEnv + "."
	overwriteTokenFlag      = "overwrite-token"
	overwriteTokenShortFlag = "o"
	overwriteTokenUsage     = "Overrides .otc-info file."
	idpNameFlag             = "idp-name"
	idpNameShortFlag        = "i"
	idpNameEnv              = "IDP_NAME"
	idpNameUsage            = "Required for authentication with IdP."
	idpUrlFlag              = "idp-url"
	idpUrlEnv               = "IDP_URL"
	idpUrlUsage             = "Required for authentication with IdP."
	totpFlag                = "totp"
	totpShortFlag           = "t"
	totpUsage               = "6-digit time-based one-time password (TOTP) used for the MFA login flow. Required together with the user-domain-id."
	userDomainIdFlag        = "os-user-domain-id"
	userDomainIdEnv         = "OS_USER_DOMAIN_ID"
	userDomainIdUsage       = "User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --totp is provided.  Either provide this argument or set the environment variable " + userDomainIdEnv + "."
)
