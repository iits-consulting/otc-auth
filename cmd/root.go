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
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"otc-auth/accesstoken"
	"otc-auth/cce"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/login"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "otc-auth",
	Short: "OTC-Auth Command Line Interface for managing OTC clouds",
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: loginCmdHelp,
}

var loginIamCmd = &cobra.Command{
	Use:     "iam",
	Short:   loginIamCmdHelp,
	Example: loginIamCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(loginIamFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		authInfo := common.AuthInfo{
			AuthType:      "iam",
			Username:      username,
			Password:      password,
			DomainName:    domainName,
			Otp:           totp,
			UserDomainID:  userDomainId,
			OverwriteFile: overwriteToken,
			Region:        region,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginIdpSamlCmd = &cobra.Command{
	Use:     "idp-saml",
	Short:   loginIdpSamlCmdHelp,
	Example: loginIdpSamlCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(loginIdpSamlOidcFlagToEnv),
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
			Region:        region,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginIdpOidcCmd = &cobra.Command{
	Use:     "idp-oidc",
	Short:   loginIdpOidcCmdHelp,
	Example: loginIdpOidcCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(loginIdpSamlOidcFlagToEnv),
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
			Region:        region,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo)
	},
}

var loginRemoveCmd = &cobra.Command{
	Use:     "remove",
	Short:   loginRemoveCmdHelp,
	Long:    "Here we can put a longer description of this command", // TODO
	Example: "Here comes an example usage of this command",          // TODO
	PreRunE: configureCmdFlagsAgainstEnvs(loginRemoveFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		config.RemoveCloudConfig(domainName)
	},
}

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: projectsCmdHelp,
}

var projectsListCmd = &cobra.Command{
	Use:     "list",
	Short:   projectsListCmdHelp,
	Example: projectsListCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		iam.GetProjectsInActiveCloud()
	},
}

var cceCmd = &cobra.Command{
	Use:               "cce",
	Short:             cceCmdHelp,
	PersistentPreRunE: configureCmdFlagsAgainstEnvs(cceListFlagToEnv),
}

var cceListCmd = &cobra.Command{
	Use:     "list",
	Short:   cceListHelp,
	Example: "", // TODO
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}
		cce.GetClusterNames(projectName)
	},
}

var cceGetKubeConfigCmd = &cobra.Command{
	Use:     "get-kube-config",
	Short:   cceGetKubeConfigHelp,
	Example: "", // TODO
	PreRunE: configureCmdFlagsAgainstEnvs(cceGetKubeConfigFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		daysValidString := strconv.Itoa(daysValid)

		if strings.HasPrefix(targetLocation, "~") {
			targetLocation = strings.Replace(targetLocation, "~", homedir.HomeDir(), 1)
		}

		kubeConfigParams := cce.KubeConfigParams{
			ProjectName:    projectName,
			ClusterName:    clusterName,
			DaysValid:      daysValidString,
			TargetLocation: targetLocation,
		}

		cce.GetKubeConfig(kubeConfigParams)
	},
}

var accessTokenCmd = &cobra.Command{
	Use:               "access-token",
	Short:             accessTokenCmdHelp,
	PersistentPreRunE: configureCmdFlagsAgainstEnvs(accessTokenFlagToEnv),
}

var accessTokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: accessTokenCreateCmdHelp,
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		accesstoken.CreateAccessToken(accessTokenCreateDescription)
	},
}

var accessTokenListCmd = &cobra.Command{
	Use:   "list",
	Short: accessTokenListCmdHelp,
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		accessTokens, err2 := accesstoken.ListAccessToken()
		if err2 != nil {
			common.OutputErrorToConsoleAndExit(err2)
		}
		if len(accessTokens) > 0 {
			log.Println("\nAccess Tokens:")
			for _, aT := range accessTokens {
				log.Printf("\nToken: \t\t%s\n"+
					"Description: \t%s\n"+
					"Created by: \t%s\n"+
					"Last Used: \t%s\n"+
					"Active: \t%s\n \n",
					aT.AccessKey, aT.Description, aT.UserID, aT.LastUseTime, aT.Status)
			}
		} else {
			log.Println("No access-tokens found")
		}
	},
}

var accessTokenDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: accessTokenDeleteCmdHelp,
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)

		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		if token == "" {
			common.OutputErrorMessageToConsoleAndExit("fatal: argument token cannot be empty.")
		}
		errDelete := accesstoken.DeleteAccessToken(token)
		if errDelete != nil {
			common.OutputErrorToConsoleAndExit(errDelete)
		}
	},
}

func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
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
	loginIamCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIamCmd.MarkFlagRequired(regionFlag)

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
	loginIdpSamlCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIdpSamlCmd.MarkFlagRequired(regionFlag)

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
	loginIdpOidcCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIdpOidcCmd.MarkFlagRequired(regionFlag)

	loginCmd.AddCommand(loginRemoveCmd)
	loginRemoveCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginRemoveCmd.MarkFlagRequired(domainNameFlag)

	RootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)

	RootCmd.AddCommand(cceCmd)
	cceCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	cceCmd.MarkPersistentFlagRequired(domainNameFlag)
	cceCmd.PersistentFlags().StringVarP(&projectName, projectNameFlag, projectNameShortFlag, "", projectNameUsage)
	cceCmd.MarkPersistentFlagRequired(projectNameFlag)

	cceCmd.AddCommand(cceListCmd)

	cceCmd.AddCommand(cceGetKubeConfigCmd)
	cceGetKubeConfigCmd.Flags().StringVarP(&clusterName, clusterNameFlag, clusterNameShortFlag, "", clusterNameUsage)
	cceGetKubeConfigCmd.MarkFlagRequired(clusterNameFlag)
	cceGetKubeConfigCmd.Flags().IntVarP(&daysValid, daysValidFlag, daysValidShortFlag, 7, daysValidUsage)
	cceGetKubeConfigCmd.MarkFlagRequired(daysValidFlag)
	cceGetKubeConfigCmd.Flags().StringVarP(&targetLocation, targetLocationFlag, targetLocationShortFlag, "~/.kube/config", targetLocationUsage)
	cceGetKubeConfigCmd.MarkFlagRequired(targetLocationFlag)

	RootCmd.AddCommand(accessTokenCmd)
	accessTokenCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	accessTokenCmd.MarkPersistentFlagRequired(domainNameFlag)

	accessTokenCmd.AddCommand(accessTokenCreateCmd)
	accessTokenCreateCmd.Flags().StringVarP(&accessTokenCreateDescription, accessTokenDescriptionFlag, accessTokenDescriptionShortFlag, "Token by otc-auth", accessTokenDescriptionUsage)
	accessTokenCreateCmd.MarkFlagRequired(accessTokenDescriptionFlag)

	accessTokenCmd.AddCommand(accessTokenListCmd)

	accessTokenCmd.AddCommand(accessTokenDeleteCmd)
	accessTokenDeleteCmd.Flags().StringVarP(&token, accessTokenTokenFlag, accessTokenTokenShortFlag, "", accessTokenTokenUsage)
	accessTokenDeleteCmd.MarkFlagRequired(accessTokenTokenFlag)
}

var (
	username                     string
	password                     string
	domainName                   string
	overwriteToken               bool
	idpName                      string
	idpUrl                       string
	totp                         string
	userDomainId                 string
	region                       string
	projectName                  string
	clusterName                  string
	daysValid                    int
	targetLocation               string
	accessTokenCreateDescription string
	token                        string

	loginIamFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIdFlag: userDomainIdEnv,
		idpNameFlag:      idpNameEnv,
		idpUrlFlag:       idpUrlEnv,
		regionFlag:       regionEnv,
	}

	loginIdpSamlOidcFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIdFlag: userDomainIdEnv,
		idpNameFlag:      idpNameEnv,
		idpUrlFlag:       idpUrlEnv,
		regionFlag:       regionEnv,
	}

	loginRemoveFlagToEnv = map[string]string{
		userDomainIdFlag: userDomainIdEnv,
	}

	cceListFlagToEnv = map[string]string{
		projectNameFlag: projectNameEnv,
		domainNameFlag:  domainNameEnv,
	}

	cceGetKubeConfigFlagToEnv = map[string]string{
		clusterNameFlag: clusterNameEnv,
	}

	accessTokenFlagToEnv = map[string]string{
		domainNameFlag: domainNameEnv,
	}
)

const (
	loginCmdHelp                    = "Login to the Open Telekom Cloud and receive an unscoped token."
	loginIamCmdHelp                 = "Login to the Open Telekom Cloud through its Identity and Access Management system and receive an unscoped token."
	loginIamCmdExample              = "otc-auth login iam --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName"
	loginIdpSamlCmdHelp             = "Login to the Open Telekom Cloud through an Identity Provider and SAML and receive an unscoped token."
	loginIdpSamlCmdExample          = "otc-auth login idp-saml --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName" // TODO: add some more examples here
	loginIdpOidcCmdHelp             = "Login to the Open Telekom Cloud through an Identity Provider and OIDC and receive an unscoped token."
	loginIdpOidcCmdExample          = "otc-auth login idp-oidc --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName" // TODO: add some more examples here
	loginRemoveCmdHelp              = "Removes login information for a cloud"
	projectsCmdHelp                 = "Manage Project Information"
	projectsListCmdHelp             = "List Projects in Active Cloud"
	projectsListCmdExample          = "otc-auth projects list"
	cceCmdHelp                      = "Manage Cloud Container Engine."
	cceListHelp                     = "Lists Project Clusters in CCE."
	cceGetKubeConfigHelp            = "Get remote kube config and merge it with existing local config file."
	accessTokenCmdHelp              = "Manage AK/SK."
	accessTokenCreateCmdHelp        = "Create new AK/SK."
	accessTokenListCmdHelp          = "List existing AK/SKs."
	accessTokenDeleteCmdHelp        = "Delete existing AK/SKs."
	usernameFlag                    = "os-username"
	usernameShortFlag               = "u"
	usernameEnv                     = "OS_USERNAME"
	usernameUsage                   = "Username for the OTC IAM system. Either provide this argument or set the environment variable " + usernameEnv + "."
	passwordFlag                    = "os-password"
	passwordShortFlag               = "p"
	passwordEnv                     = "OS_PASSWORD"
	passwordUsage                   = "Password for the OTC IAM system. Either provide this argument or set the environment variable " + passwordEnv + "."
	domainNameFlag                  = "os-domain-name"
	domainNameShortFlag             = "d"
	domainNameEnv                   = "OS_DOMAIN_NAME"
	domainNameUsage                 = "OTC domain name. Either provide this argument or set the environment variable " + domainNameEnv + "."
	overwriteTokenFlag              = "overwrite-token"
	overwriteTokenShortFlag         = "o"
	overwriteTokenUsage             = "Overrides .otc-info file."
	idpNameFlag                     = "idp-name"
	idpNameShortFlag                = "i"
	idpNameEnv                      = "IDP_NAME"
	idpNameUsage                    = "Required for authentication with IdP."
	idpUrlFlag                      = "idp-url"
	idpUrlEnv                       = "IDP_URL"
	idpUrlUsage                     = "Required for authentication with IdP."
	totpFlag                        = "totp"
	totpShortFlag                   = "t"
	totpUsage                       = "6-digit time-based one-time password (TOTP) used for the MFA login flow. Required together with the user-domain-id."
	userDomainIdFlag                = "os-user-domain-id"
	userDomainIdEnv                 = "OS_USER_DOMAIN_ID"
	userDomainIdUsage               = "User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --totp is provided.  Either provide this argument or set the environment variable " + userDomainIdEnv + "."
	regionFlag                      = "region"
	regionShortFlag                 = "r"
	regionEnv                       = "REGION"
	regionUsage                     = "OTC region code. Either provide this argument or set the environment variable " + regionEnv + "." // TODO: fill out the region
	projectNameFlag                 = "os-project-name"
	projectNameShortFlag            = "p"
	projectNameEnv                  = "OS_PROJECT_NAME"
	projectNameUsage                = "Name of the project you want to access. Either provide this argument or set the environment variable " + projectNameEnv + "."
	clusterNameFlag                 = "cluster"
	clusterNameShortFlag            = "c"
	clusterNameEnv                  = "CLUSTER_NAME"
	clusterNameUsage                = "Name of the clusterArg you want to access. Either provide this argument or set the environment variable " + clusterNameEnv + "."
	daysValidFlag                   = "days-valid"
	daysValidShortFlag              = "v"
	daysValidUsage                  = "Period (in days) that the config will be valid."
	targetLocationFlag              = "target-location"
	targetLocationShortFlag         = "l"
	targetLocationUsage             = "Where the kube config should be saved"
	accessTokenDescriptionFlag      = "description"
	accessTokenDescriptionShortFlag = "s"
	accessTokenDescriptionUsage     = "Description of the token"
	accessTokenTokenFlag            = "token"
	accessTokenTokenShortFlag       = "t"
	accessTokenTokenUsage           = "The AK/SK token to delete."
)
