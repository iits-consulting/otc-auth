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
//nolint:gochecknoglobals // Globals are used to make the parsing and reuseability of the cmd functionality easier
package cmd

import (
	"log"
	"os"
	"strconv"
	"strings"

	"otc-auth/accesstoken"
	"otc-auth/cce"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/login"
	"otc-auth/openstack"

	"github.com/spf13/cobra"
	"k8s.io/client-go/util/homedir"
)

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
			UserDomainID:  userDomainID,
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
			IdpURL:        idpURL,
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
			IdpURL:        idpURL,
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
	Example: loginRemoveCmdExample,
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
	Short:   cceListCmdHelp,
	Example: cceListCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.",
			)
		}
		cce.GetClusterNames(projectName)
	},
}

var cceGetKubeConfigCmd = &cobra.Command{
	Use:     "get-kube-config",
	Short:   cceGetKubeConfigCmdHelp,
	Example: cceGetKubeConfigCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(cceGetKubeConfigFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.",
			)
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

var tempAccessTokenCmd = &cobra.Command{
	Use:               "temp-access-token",
	Short:             accessTokenCmdHelp,
	PersistentPreRunE: configureCmdFlagsAgainstEnvs(accessTokenFlagToEnv),
}

var tempAccessTokenCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   tempAccessTokenCreateCmdHelp,
	Example: tempAccessTokenCreateCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		config.LoadCloudConfig(domainName)
		if !config.IsAuthenticationValid() {
			common.OutputErrorMessageToConsoleAndExit(
				"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first.")
		}

		accesstoken.CreateTemporaryAccessToken(temporaryAccessTokenDurationSeconds)
	},
}

var accessTokenCmd = &cobra.Command{
	Use:               "access-token",
	Short:             accessTokenCmdHelp,
	PersistentPreRunE: configureCmdFlagsAgainstEnvs(accessTokenFlagToEnv),
}

var accessTokenCreateCmd = &cobra.Command{
	Use:     "create",
	Short:   accessTokenCreateCmdHelp,
	Example: accessTokenCreateCmdExample,
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
	Use:     "delete",
	Short:   accessTokenDeleteCmdHelp,
	Example: accessTokenDeleteCmdExample,
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

var openstackCmd = &cobra.Command{
	Use:   "openstack",
	Short: openstackCmdHelp,
}

var openstackConfigCreateCmd = &cobra.Command{
	Use:   "config-create",
	Short: openstackConfigCreateCmdHelp,
	Run: func(cmd *cobra.Command, args []string) {
		if strings.HasPrefix(openStackConfigLocation, "~") {
			openStackConfigLocation = strings.Replace(openStackConfigLocation, "~", homedir.HomeDir(), 1)
		}
		openstack.WriteOpenStackCloudsYaml(openStackConfigLocation)
	},
}

func Execute() {
	setupRootCmd()
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

//nolint:errcheck,funlen // error never occurs, setup has to be that lengthy
func setupRootCmd() {
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
	loginIamCmd.Flags().StringVarP(&userDomainID, userDomainIDFlag, "", "", userDomainIDUsage)
	loginIamCmd.MarkFlagsRequiredTogether(totpFlag, userDomainIDFlag)
	loginIamCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIamCmd.MarkFlagRequired(regionFlag)

	loginCmd.AddCommand(loginIdpSamlCmd)
	loginIdpSamlCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIdpSamlCmd.MarkFlagRequired(usernameFlag)
	loginIdpSamlCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIdpSamlCmd.MarkFlagRequired(passwordFlag)
	loginIdpSamlCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpSamlCmd.MarkFlagRequired(domainNameFlag)
	loginIdpSamlCmd.Flags().BoolVarP(
		&overwriteToken,
		overwriteTokenFlag,
		overwriteTokenShortFlag,
		false,
		overwriteTokenUsage,
	)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpSamlCmd.MarkPersistentFlagRequired(idpNameFlag)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpURL, idpURLFlag, "", "", idpURLUsage)
	loginIdpSamlCmd.MarkPersistentFlagRequired(idpURLFlag)
	loginIdpSamlCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIdpSamlCmd.MarkFlagRequired(regionFlag)

	loginCmd.AddCommand(loginIdpOidcCmd)
	loginIdpOidcCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIdpOidcCmd.MarkFlagRequired(usernameFlag)
	loginIdpOidcCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIdpOidcCmd.MarkFlagRequired(passwordFlag)
	loginIdpOidcCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpOidcCmd.MarkFlagRequired(domainNameFlag)
	loginIdpOidcCmd.Flags().BoolVarP(
		&overwriteToken,
		overwriteTokenFlag,
		overwriteTokenShortFlag,
		false,
		overwriteTokenUsage,
	)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpOidcCmd.MarkPersistentFlagRequired(idpNameFlag)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpURL, idpURLFlag, "", "", idpURLUsage)
	loginIdpOidcCmd.MarkPersistentFlagRequired(idpURLFlag)
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
	cceGetKubeConfigCmd.Flags().IntVarP(
		&daysValid,
		daysValidFlag,
		daysValidShortFlag,
		daysValidDefaultValue,
		daysValidUsage,
	)
	cceGetKubeConfigCmd.MarkFlagRequired(daysValidFlag)
	cceGetKubeConfigCmd.Flags().StringVarP(
		&targetLocation,
		targetLocationFlag,
		targetLocationShortFlag,
		"~/.kube/config",
		targetLocationUsage,
	)
	cceGetKubeConfigCmd.MarkFlagRequired(targetLocationFlag)

	RootCmd.AddCommand(tempAccessTokenCmd)
	tempAccessTokenCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	tempAccessTokenCmd.MarkPersistentFlagRequired(domainNameFlag)

	tempAccessTokenCmd.AddCommand(tempAccessTokenCreateCmd)
	tempAccessTokenCmd.Flags().IntVarP(
		&temporaryAccessTokenDurationSeconds,
		temporaryAccessTokenDurationSecondsFlag,
		temporaryAccessTokenDurationSecondsShortFlag,
		15*60, // default is 15 minutes
		temporaryAccessTokenDurationSecondsUsage,
	)

	RootCmd.AddCommand(accessTokenCmd)
	accessTokenCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	accessTokenCmd.MarkPersistentFlagRequired(domainNameFlag)

	accessTokenCmd.AddCommand(accessTokenCreateCmd)
	accessTokenCreateCmd.Flags().StringVarP(
		&accessTokenCreateDescription,
		accessTokenDescriptionFlag,
		accessTokenDescriptionShortFlag,
		"Token by otc-auth",
		accessTokenDescriptionUsage,
	)
	accessTokenCreateCmd.MarkFlagRequired(accessTokenDescriptionFlag)

	accessTokenCmd.AddCommand(accessTokenListCmd)

	accessTokenCmd.AddCommand(accessTokenDeleteCmd)
	accessTokenDeleteCmd.Flags().StringVarP(
		&token,
		accessTokenTokenFlag,
		accessTokenTokenShortFlag,
		"",
		accessTokenTokenUsage,
	)
	accessTokenDeleteCmd.MarkFlagRequired(accessTokenTokenFlag)

	RootCmd.AddCommand(openstackCmd)
	openstackCmd.AddCommand(openstackConfigCreateCmd)
	openstackConfigCreateCmd.Flags().StringVarP(
		&openStackConfigLocation,
		openstackConfigCreateConfigLocationFlag,
		openstackConfigCreateConfigLocationShortFlag,
		"~/.config/openstack/clouds.yaml",
		openstackConfigCreateConfigLocationUsage,
	)
}

var (
	username                            string
	password                            string
	domainName                          string
	overwriteToken                      bool
	idpName                             string
	idpURL                              string
	totp                                string
	userDomainID                        string
	region                              string
	projectName                         string
	clusterName                         string
	daysValid                           int
	targetLocation                      string
	accessTokenCreateDescription        string
	temporaryAccessTokenDurationSeconds int
	token                               string
	openStackConfigLocation             string

	loginIamFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIDFlag: userDomainIDEnv,
		idpNameFlag:      idpNameEnv,
		idpURLFlag:       idpURLEnv,
		regionFlag:       regionEnv,
	}

	loginIdpSamlOidcFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userDomainIDFlag: userDomainIDEnv,
		idpNameFlag:      idpNameEnv,
		idpURLFlag:       idpURLEnv,
		regionFlag:       regionEnv,
	}

	loginRemoveFlagToEnv = map[string]string{
		userDomainIDFlag: userDomainIDEnv,
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

//nolint:lll // The long lines represent
const (
	loginCmdHelp       = "Login to the Open Telekom Cloud and receive an unscoped token."
	loginIamCmdHelp    = "Login to the Open Telekom Cloud through its Identity and Access Management system and receive an unscoped token."
	loginIamCmdExample = `$ otc-auth login iam --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName

$ export OS_USERNAME=YourUsername
$ export OS_PASSWORD=YourPassword
$ export OS_DOMAIN_NAME=YourDomainName
$ otc-auth login iam

$ export OS_USERNAME=YourUsername
$ export OS_PASSWORD=YourPassword
$ export OS_DOMAIN_NAME=YourDomainName
$ export REGION=YourRegion
$ otc-auth login iam --overwrite-token --region YourRegion`
	loginIdpSamlCmdHelp    = "Login to the Open Telekom Cloud through an Identity Provider and SAML and receive an unscoped token."
	loginIdpSamlCmdExample = `otc-auth login idp-saml --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName

export OS_DOMAIN_NAME=MyDomain
export OS_USERNAME=MyUsername
export OS_PASSWORD=MyPassword
export REGION=MyRegion
otc-auth login idp-saml --idp-name MyIdP --idp-url https://example.com/saml

export OS_DOMAIN_NAME=MyDomain
export OS_PASSWORD=MyPassword
otc-auth login idp-saml --idp-name MyIdP --idp-url https://example.com/saml --os-username MyUsername --region MyRegion`
	loginIdpOidcCmdHelp    = "Login to the Open Telekom Cloud through an Identity Provider and OIDC and receive an unscoped token."
	loginIdpOidcCmdExample = `otc-auth login idp-oidc --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName

export OS_DOMAIN_NAME=MyDomain
export OS_USERNAME=MyUsername
export OS_PASSWORD=MyPassword
export REGION=MyRegion
otc-auth login idp-oidc --idp-name MyIdP --idp-url https://example.com/oidc

export OS_DOMAIN_NAME=MyDomain
export OS_PASSWORD=MyPassword
otc-auth login idp-oidc --idp-name MyIdP --idp-url https://example.com/oidc --os-username MyUsername --region MyRegion`
	loginRemoveCmdHelp    = "Removes login information for a cloud"
	loginRemoveCmdExample = `$ otc-auth login remove --os-domain-name MyLogin

$ export OS_DOMAIN_NAME=MyLogin
$ otc-auth login remove`
	projectsCmdHelp        = "Manage Project Information"
	projectsListCmdHelp    = "List Projects in Active Cloud"
	projectsListCmdExample = "otc-auth projects list"
	cceCmdHelp             = "Manage Cloud Container Engine."
	cceListCmdHelp         = "Lists Project Clusters in CCE."
	cceListCmdExample      = `$ otc-auth cce list --os-project-name MyProject

$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce list

$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce list`
	cceGetKubeConfigCmdHelp    = "Get remote kube config and merge it with existing local config file."
	cceGetKubeConfigCmdExample = `$ otc-auth cce get-kube-config --cluster MyCluster --target-location /path/to/config

$ export CLUSTER_NAME=MyCluster
$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce get-kube-config --days-valid 14

$ export CLUSTER_NAME=MyCluster
$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce get-kube-config`

	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenCmdHelp = "Manage AK/SK."
	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenCreateCmdHelp = "Create new AK/SK."

	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenCreateCmdExample = `$ otc-auth access-token create --description "Custom token description"

$ otc-auth access-token create

$ export OS_DOMAIN_NAME=MyDomain
$ otc-auth access-token create`
	accessTokenListCmdHelp   = "List existing AK/SKs."
	accessTokenDeleteCmdHelp = "Delete existing AK/SKs."
	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenDeleteCmdExample = `$ otc-auth access-token delete --token YourToken

$ export OS_DOMAIN_NAME=YourDomain
$ export AK_SK_TOKEN=YourToken
$ otc-auth access-token delete

$ otc-auth access-token delete --token YourToken --os-domain-name YourDomain`
	tempAccessTokenCreateCmdExample = `$ otc-auth temp-access-token create -t 900 -d YourDomainName # this creates a temp AK/SK which is 15 minutes valid (15 * 60 = 900)
	
	$ otc-auth temp-access-token create --duration-seconds 1800`
	openstackCmdHelp             = "Manage Openstack Integration"
	openstackConfigCreateCmdHelp = "Creates new clouds.yaml"
	usernameFlag                 = "os-username"
	usernameShortFlag            = "u"
	usernameEnv                  = "OS_USERNAME"
	usernameUsage                = "Username for the OTC IAM system. Either provide this argument or set the environment variable " + usernameEnv + "."
	passwordFlag                 = "os-password"
	passwordShortFlag            = "p"
	passwordEnv                  = "OS_PASSWORD"
	passwordUsage                = "Password for the OTC IAM system. Either provide this argument or set the environment variable " + passwordEnv + "."
	domainNameFlag               = "os-domain-name"
	domainNameShortFlag          = "d"
	domainNameEnv                = "OS_DOMAIN_NAME"
	domainNameUsage              = "OTC domain name. Either provide this argument or set the environment variable " + domainNameEnv + "."
	overwriteTokenFlag           = "overwrite-token"
	overwriteTokenShortFlag      = "o"
	//nolint:gosec // This is not a hardcoded credential but a help message with a filename inside
	overwriteTokenUsage                          = "Overrides .otc-info file."
	idpNameFlag                                  = "idp-name"
	idpNameShortFlag                             = "i"
	idpNameEnv                                   = "IDP_NAME"
	idpNameUsage                                 = "Required for authentication with IdP."
	idpURLFlag                                   = "idp-url"
	idpURLEnv                                    = "IDP_URL"
	idpURLUsage                                  = "Required for authentication with IdP."
	totpFlag                                     = "totp"
	totpShortFlag                                = "t"
	totpUsage                                    = "6-digit time-based one-time password (TOTP) used for the MFA login flow. Required together with the user-domain-id."
	userDomainIDFlag                             = "os-user-domain-id"
	userDomainIDEnv                              = "OS_USER_DOMAIN_ID"
	userDomainIDUsage                            = "User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --totp is provided.  Either provide this argument or set the environment variable " + userDomainIDEnv + "."
	regionFlag                                   = "region"
	regionShortFlag                              = "r"
	regionEnv                                    = "REGION"
	regionUsage                                  = "OTC region code. Either provide this argument or set the environment variable " + regionEnv + "."
	projectNameFlag                              = "os-project-name"
	projectNameShortFlag                         = "p"
	projectNameEnv                               = "OS_PROJECT_NAME"
	projectNameUsage                             = "Name of the project you want to access. Either provide this argument or set the environment variable " + projectNameEnv + "."
	clusterNameFlag                              = "cluster"
	clusterNameShortFlag                         = "c"
	clusterNameEnv                               = "CLUSTER_NAME"
	clusterNameUsage                             = "Name of the clusterArg you want to access. Either provide this argument or set the environment variable " + clusterNameEnv + "."
	daysValidFlag                                = "days-valid"
	daysValidDefaultValue                        = 7
	daysValidShortFlag                           = "v"
	daysValidUsage                               = "Period (in days) that the config will be valid."
	targetLocationFlag                           = "target-location"
	targetLocationShortFlag                      = "l"
	targetLocationUsage                          = "Where the kube config should be saved"
	accessTokenDescriptionFlag                   = "description"
	accessTokenDescriptionShortFlag              = "s"
	accessTokenDescriptionUsage                  = "Description of the token"
	accessTokenTokenFlag                         = "token"
	accessTokenTokenShortFlag                    = "t"
	tempAccessTokenCreateCmdHelp                 = "Manage temporary AK/SK"
	temporaryAccessTokenDurationSecondsFlag      = "duration-seconds"
	temporaryAccessTokenDurationSecondsShortFlag = "t"
	temporaryAccessTokenDurationSecondsUsage     = "The token's lifetime, in seconds. Valid times are between 900 and 86400 seconds."
	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenTokenUsage                        = "The AK/SK token to delete."
	openstackConfigCreateConfigLocationFlag      = "config-location"
	openstackConfigCreateConfigLocationShortFlag = "l"
	openstackConfigCreateConfigLocationUsage     = "Where the config should be saved."
)
