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
	"errors"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"

	"otc-auth/accesstoken"
	"otc-auth/cce"
	"otc-auth/common"
	"otc-auth/config"
	"otc-auth/iam"
	"otc-auth/login"
	"otc-auth/openstack"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var RootCmd = &cobra.Command{
	Use:     "otc-auth",
	Short:   "OTC-Auth Command Line Interface for managing OTC clouds",
	PreRunE: configureCmdFlagsAgainstEnvs(rootFlagToEnv),
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
		if totp != "" && username != "" {
			common.ThrowError(errors.New("when using MFA (totp), the userID should be given, not the username"))
		}
		if (userID != "" && username != "") || (userID == "" && username == "") {
			common.ThrowError(errors.New("either the username or the userID must be set, not both"))
		}
		authInfo := common.AuthInfo{
			AuthType:      "iam",
			Username:      username,
			Password:      password,
			DomainName:    domainName,
			Otp:           totp,
			UserID:        userID,
			OverwriteFile: overwriteToken,
			Region:        region,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
	},
}

var loginIdpSamlCmd = &cobra.Command{
	Use:     "idp-saml",
	Short:   loginIdpSamlCmdHelp,
	Example: loginIdpSamlCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(loginIdpSamlFlagToEnv),
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
		login.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
	},
}

var loginIdpOidcCmd = &cobra.Command{
	Use:     "idp-oidc",
	Short:   loginIdpOidcCmdHelp,
	Example: loginIdpOidcCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(loginIdpOidcFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		authInfo := common.AuthInfo{
			AuthType:         "idp",
			ClientID:         clientID,
			ClientSecret:     clientSecret,
			DomainName:       domainName,
			IdpName:          idpName,
			IdpURL:           idpURL,
			AuthProtocol:     "oidc",
			OverwriteFile:    overwriteToken,
			Region:           region,
			OidcScopes:       oidcScopes,
			IsServiceAccount: isServiceAccount,
		}
		login.AuthenticateAndGetUnscopedToken(authInfo, skipTLS)
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
	PersistentPreRunE: configureCmdFlagsAgainstEnvs(cceFlagToEnv),
}

var cceListCmd = &cobra.Command{
	Use:     "list",
	Short:   cceListCmdHelp,
	Example: cceListCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(cceListFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}
		if !config.IsAuthenticationValid() {
			common.ThrowError(
				errors.New("fatal: no valid unscoped token found." +
					"\n\nPlease obtain an unscoped token by logging in first"))
		}
		cce.GetClusterNames(projectName)
	},
}

var cceCheckKubeConfigCmd = &cobra.Command{
	Use:     "check-kube-certs",
	Short:   cceCheckKubeCertsCmdHelp,
	Example: cceCheckKubeCertsCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		currentConfig, err := clientcmd.NewDefaultClientConfigLoadingRules().GetStartingConfig()
		if err != nil {
			common.ThrowError(err)
		}
		if currentConfig == nil {
			common.ThrowError(errors.New(""))
		}
		cce.CheckAndWarnCertsValidity(*currentConfig)
	},
}

var cceGetKubeConfigCmd = &cobra.Command{
	Use:     "get-kube-config",
	Short:   cceGetKubeConfigCmdHelp,
	Example: cceGetKubeConfigCmdExample,
	PreRunE: configureCmdFlagsAgainstEnvs(cceGetKubeConfigFlagToEnv),
	Run: func(cmd *cobra.Command, args []string) {
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}
		if !config.IsAuthenticationValid() {
			common.ThrowError(
				errors.New("fatal: no valid unscoped token found." +
					"\n\nPlease obtain an unscoped token by logging in first"))
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
			Server:         server,
		}

		cce.GetKubeConfig(kubeConfigParams, skipKubeTLS, printKubeConfig, alias)
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
	RunE: func(cmd *cobra.Command, args []string) error {
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}
		if !config.IsAuthenticationValid() {
			return errors.New(
				"fatal: no valid unscoped token found, please obtain an unscoped token by logging in first",
			)
		}

		if temporaryAccessTokenDurationSeconds < 900 || temporaryAccessTokenDurationSeconds > 86400 {
			return errors.New("fatal: token duration must be between 900 and 86400 seconds (15m and 24h)")
		}
		err = accesstoken.CreateTemporaryAccessToken(temporaryAccessTokenDurationSeconds, printAkSk)
		if err != nil {
			return err
		}
		return nil
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
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}
		if !config.IsAuthenticationValid() {
			common.ThrowError(
				errors.New(
					"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first"))
		}

		accesstoken.CreateAccessToken(accessTokenCreateDescription, printAkSk)
	},
}

var accessTokenListCmd = &cobra.Command{
	Use:   "list",
	Short: accessTokenListCmdHelp,
	Run: func(cmd *cobra.Command, args []string) {
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}
		if !config.IsAuthenticationValid() {
			common.ThrowError(
				errors.New("fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first"))
		}

		accessTokens, errListToken := accesstoken.ListAccessToken()
		if errListToken != nil {
			common.ThrowError(errListToken)
		}
		if len(accessTokens) > 0 {
			output := "\nAccess Tokens:"
			for _, aT := range accessTokens {
				output += fmt.Sprintf("\nToken: \t\t%s\n"+
					"Description: \t%s\n"+
					"Created by: \t%s\n"+
					"Last Used: \t%s\n"+
					"Active: \t%s\n \n",
					aT.AccessKey, aT.Description, aT.UserID, aT.LastUseTime, aT.Status)
			}
			_, wErr := log.Writer().Write([]byte(output))
			if wErr != nil {
				common.ThrowError(fmt.Errorf("fatal: couldn't write output: %w", wErr))
			}
		} else {
			glog.V(1).Info("info: no access-tokens found")
		}
	},
}

var accessTokenDeleteCmd = &cobra.Command{
	Use:     "delete",
	Short:   accessTokenDeleteCmdHelp,
	Example: accessTokenDeleteCmdExample,
	Run: func(cmd *cobra.Command, args []string) {
		err := config.LoadCloudConfig(domainName)
		if err != nil {
			common.ThrowError(errors.New("fatal: couldn't load cloud config: " + err.Error()))
		}

		if !config.IsAuthenticationValid() {
			common.ThrowError(
				errors.New(
					"fatal: no valid unscoped token found.\n\nPlease obtain an unscoped token by logging in first"))
		}

		if token == "" {
			common.ThrowError(errors.New("fatal: argument token cannot be empty"))
		}
		errDelete := accesstoken.DeleteAccessToken(token)
		if errDelete != nil {
			common.ThrowError(errDelete)
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
	// Parse glog flags first
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	setupRootCmd()
	err := RootCmd.Execute()
	if err != nil {
		glog.Exitf("fatal: error executing root cmd: %v", err)
	}
}

//nolint:funlen // setup has to be that lengthy
func setupRootCmd() {
	RootCmd.AddCommand(loginCmd)
	RootCmd.PersistentFlags().BoolVarP(&skipTLS, skipTLSFlag, skipTLSShortFlag, false, skipTLSUsage)

	loginCmd.AddCommand(loginIamCmd)
	loginIamCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIamCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIamCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIamCmd.Flags().BoolVarP(&overwriteToken, overwriteTokenFlag, overwriteTokenShortFlag, false, overwriteTokenUsage)
	loginIamCmd.Flags().StringVarP(&totp, totpFlag, totpShortFlag, "", totpUsage)
	loginIamCmd.Flags().StringVarP(&userID, userIDFlag, "", "", userIDUsage)
	loginIamCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)

	loginCmd.AddCommand(loginIdpSamlCmd)
	loginIdpSamlCmd.Flags().StringVarP(&username, usernameFlag, usernameShortFlag, "", usernameUsage)
	loginIdpSamlCmd.Flags().StringVarP(&password, passwordFlag, passwordShortFlag, "", passwordUsage)
	loginIdpSamlCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpSamlCmd.Flags().BoolVarP(
		&overwriteToken,
		overwriteTokenFlag,
		overwriteTokenShortFlag,
		false,
		overwriteTokenUsage,
	)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpSamlCmd.PersistentFlags().StringVarP(&idpURL, idpURLFlag, "", "", idpURLUsage)
	loginIdpSamlCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)

	loginCmd.AddCommand(loginIdpOidcCmd)
	loginIdpOidcCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	loginIdpOidcCmd.Flags().BoolVarP(
		&overwriteToken,
		overwriteTokenFlag,
		overwriteTokenShortFlag,
		false,
		overwriteTokenUsage,
	)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpName, idpNameFlag, idpNameShortFlag, "", idpNameUsage)
	loginIdpOidcCmd.PersistentFlags().StringVarP(&idpURL, idpURLFlag, "", "", idpURLUsage)
	loginIdpOidcCmd.Flags().StringVarP(&region, regionFlag, regionShortFlag, "", regionUsage)
	loginIdpOidcCmd.Flags().StringVarP(&clientSecret, clientSecretFlag, clientSecretShortFlag, "", clientSecretUsage)
	loginIdpOidcCmd.Flags().StringVarP(&clientID, clientIDFlag, clientIDShortFlag, "", clientIDUsage)
	loginIdpOidcCmd.Flags().StringSliceVarP(&oidcScopes, oidcScopesFlag, oidcScopesShortFlag,
		[]string{"openid"}, oidcScopesUsage)
	loginIdpOidcCmd.Flags().BoolVarP(&isServiceAccount, isServiceAccountFlag, isServiceAccountShortFlag, false,
		isServiceAccountUsage)

	loginCmd.AddCommand(loginRemoveCmd)
	loginRemoveCmd.Flags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)

	RootCmd.AddCommand(projectsCmd)
	projectsCmd.AddCommand(projectsListCmd)

	RootCmd.AddCommand(cceCmd)
	cceCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	cceCmd.PersistentFlags().StringVarP(&projectName, projectNameFlag, projectNameShortFlag, "", projectNameUsage)

	cceCmd.AddCommand(cceListCmd)

	cceCmd.AddCommand(cceGetKubeConfigCmd)
	cceGetKubeConfigCmd.Flags().BoolVarP(&printKubeConfig, printKubeConfigFlag, printKubeConfigShortFlag,
		false, printKubeConfigUsage)
	cceGetKubeConfigCmd.Flags().StringVarP(&alias, aliasFlag, aliasShortFlag, "", aliasUsage)
	cceGetKubeConfigCmd.Flags().StringVarP(&clusterName, clusterNameFlag, clusterNameShortFlag, "", clusterNameUsage)
	cceGetKubeConfigCmd.Flags().BoolVarP(&skipKubeTLS, skipKubeTLSFlag, "", false, skipKubeTLSUsage)
	cceGetKubeConfigCmd.Flags().IntVarP(
		&daysValid,
		daysValidFlag,
		"",
		daysValidDefaultValue,
		daysValidUsage,
	)
	cceGetKubeConfigCmd.Flags().StringVarP(
		&server,
		serverFlag,
		serverShortFlag,
		"",
		serverUsage)
	cceGetKubeConfigCmd.Flags().StringVarP(
		&targetLocation,
		targetLocationFlag,
		targetLocationShortFlag,
		"~/.kube/config",
		targetLocationUsage,
	)
	cceCmd.AddCommand(cceCheckKubeConfigCmd)

	RootCmd.AddCommand(tempAccessTokenCmd)
	tempAccessTokenCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	tempAccessTokenCmd.AddCommand(tempAccessTokenCreateCmd)
	tempAccessTokenCreateCmd.Flags().IntVarP(
		&temporaryAccessTokenDurationSeconds,
		temporaryAccessTokenDurationSecondsFlag,
		temporaryAccessTokenDurationSecondsShortFlag,
		tempAccessTokenLifetime,
		temporaryAccessTokenDurationSecondsUsage,
	)
	tempAccessTokenCreateCmd.Flags().BoolVarP(&printAkSk, printAkSkFlag, printAkSkShortFlag,
		false, printAkSkUsage)
	RootCmd.AddCommand(accessTokenCmd)
	accessTokenCmd.PersistentFlags().StringVarP(&domainName, domainNameFlag, domainNameShortFlag, "", domainNameUsage)
	accessTokenCmd.AddCommand(accessTokenCreateCmd)
	accessTokenCreateCmd.Flags().StringVarP(
		&accessTokenCreateDescription,
		accessTokenDescriptionFlag,
		accessTokenDescriptionShortFlag,
		"Token by otc-auth",
		accessTokenDescriptionUsage,
	)
	accessTokenCreateCmd.Flags().BoolVarP(&printAkSk, printAkSkFlag, printAkSkShortFlag,
		false, printAkSkUsage)

	accessTokenCmd.AddCommand(accessTokenListCmd)
	accessTokenCmd.AddCommand(accessTokenDeleteCmd)
	accessTokenDeleteCmd.Flags().StringVarP(
		&token,
		accessTokenTokenFlag,
		accessTokenTokenShortFlag,
		"",
		accessTokenTokenUsage,
	)

	RootCmd.AddCommand(openstackCmd)
	openstackCmd.AddCommand(openstackConfigCreateCmd)
	openstackConfigCreateCmd.Flags().StringVarP(
		&openStackConfigLocation,
		openstackConfigCreateConfigLocationFlag,
		openstackConfigCreateConfigLocationShortFlag,
		"~/.config/openstack/clouds.yaml",
		openstackConfigCreateConfigLocationUsage,
	)

	cobra.CheckErr(errors.Join(
		loginIamCmd.MarkFlagRequired(passwordFlag),
		loginIamCmd.MarkFlagRequired(domainNameFlag),
		loginIamCmd.MarkFlagRequired(regionFlag),
		loginIdpSamlCmd.MarkFlagRequired(usernameFlag),
		loginIdpSamlCmd.MarkFlagRequired(passwordFlag),
		loginIdpSamlCmd.MarkFlagRequired(domainNameFlag),
		loginIdpSamlCmd.MarkPersistentFlagRequired(idpNameFlag),
		loginIdpSamlCmd.MarkFlagRequired(regionFlag),
		loginIdpOidcCmd.MarkFlagRequired(domainNameFlag),
		loginIdpSamlCmd.MarkPersistentFlagRequired(idpURLFlag),
		loginIdpOidcCmd.MarkPersistentFlagRequired(idpNameFlag),
		loginIdpOidcCmd.MarkPersistentFlagRequired(idpURLFlag),
		loginIdpOidcCmd.MarkFlagRequired(regionFlag),
		loginIdpOidcCmd.MarkFlagRequired(clientIDFlag),
		loginRemoveCmd.MarkFlagRequired(domainNameFlag),
		cceCmd.MarkPersistentFlagRequired(domainNameFlag),
		cceCmd.MarkPersistentFlagRequired(projectNameFlag),
		cceGetKubeConfigCmd.MarkFlagRequired(clusterNameFlag),
		tempAccessTokenCmd.MarkPersistentFlagRequired(domainNameFlag),
		accessTokenCmd.MarkPersistentFlagRequired(domainNameFlag),
		accessTokenDeleteCmd.MarkFlagRequired(accessTokenTokenFlag),
	))
}

var (
	username                            string
	password                            string
	domainName                          string
	overwriteToken                      bool
	idpName                             string
	idpURL                              string
	totp                                string
	userID                              string
	region                              string
	skipKubeTLS                         bool
	projectName                         string
	clusterName                         string
	daysValid                           int
	targetLocation                      string
	server                              string
	accessTokenCreateDescription        string
	temporaryAccessTokenDurationSeconds int
	token                               string
	openStackConfigLocation             string
	skipTLS                             bool
	printKubeConfig                     bool
	alias                               string
	clientSecret                        string
	clientID                            string
	oidcScopes                          []string
	printAkSk                           bool
	isServiceAccount                    bool

	rootFlagToEnv = map[string]string{
		skipTLSFlag: skipTLSEnv,
	}

	loginIamFlagToEnv = map[string]string{
		usernameFlag:   usernameEnv,
		passwordFlag:   passwordEnv,
		domainNameFlag: domainNameEnv,
		userIDFlag:     userIDEnv,
		idpNameFlag:    idpNameEnv,
		idpURLFlag:     idpURLEnv,
		regionFlag:     regionEnv,
	}

	loginIdpSamlFlagToEnv = map[string]string{
		usernameFlag:   usernameEnv,
		passwordFlag:   passwordEnv,
		domainNameFlag: domainNameEnv,
		userIDFlag:     userIDEnv,
		idpNameFlag:    idpNameEnv,
		idpURLFlag:     idpURLEnv,
		regionFlag:     regionEnv,
	}

	loginIdpOidcFlagToEnv = map[string]string{
		usernameFlag:     usernameEnv,
		passwordFlag:     passwordEnv,
		domainNameFlag:   domainNameEnv,
		userIDFlag:       userIDEnv,
		idpNameFlag:      idpNameEnv,
		idpURLFlag:       idpURLEnv,
		regionFlag:       regionEnv,
		clientIDFlag:     clientIDEnv,
		clientSecretFlag: clientSecretEnv,
		oidcScopesFlag:   oidcScopesEnv,
	}

	loginRemoveFlagToEnv = map[string]string{
		userIDFlag: userIDEnv,
	}

	cceFlagToEnv = map[string]string{
		projectNameFlag: projectNameEnv,
		domainNameFlag:  domainNameEnv,
	}

	cceListFlagToEnv = map[string]string{
		regionFlag: regionEnv,
	}

	cceGetKubeConfigFlagToEnv = map[string]string{
		clusterNameFlag: clusterNameEnv,
		regionFlag:      regionEnv,
	}

	accessTokenFlagToEnv = map[string]string{
		domainNameFlag: domainNameEnv,
	}
)

//nolint:lll // Long lines required for formatting reasons
const (
	loginCmdHelp       = "Login to the Open Telekom Cloud and receive an unscoped token"
	loginIamCmdHelp    = "Login to the Open Telekom Cloud through its Identity and Access Management system and receive an unscoped token"
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
	loginIdpSamlCmdHelp    = "Login to the Open Telekom Cloud through an Identity Provider and SAML and receive an unscoped token"
	loginIdpSamlCmdExample = `otc-auth login idp-saml --os-username YourUsername --os-password YourPassword --os-domain-name YourDomainName

export OS_DOMAIN_NAME=MyDomain
export OS_USERNAME=MyUsername
export OS_PASSWORD=MyPassword
export REGION=MyRegion
otc-auth login idp-saml --idp-name MyIdP --idp-url https://example.com/saml

export OS_DOMAIN_NAME=MyDomain
export OS_PASSWORD=MyPassword
otc-auth login idp-saml --idp-name MyIdP --idp-url https://example.com/saml --os-username MyUsername --region MyRegion`
	loginIdpOidcCmdHelp    = "Login to the Open Telekom Cloud through an Identity Provider and OIDC and receive an unscoped token"
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
	cceCmdHelp             = "Manage Cloud Container Engine"
	cceListCmdHelp         = "Lists Project Clusters in CCE"
	cceListCmdExample      = `$ otc-auth cce list --os-project-name MyProject

$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce list

$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce list`
	cceCheckKubeCertsCmdHelp    = "Reads KubeConfig and warns when malformed or expired certificates are found.\nThis does NOT check to make sure certs are correctly signed or that the hostnames are correct for your usecase."
	cceCheckKubeCertsCmdExample = `$ otc-auth cce check-kube-certs --os-project-name MyProject

$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce check-kube-certs
`
	cceGetKubeConfigCmdHelp    = "Get remote kube config and merge it with existing local config file"
	cceGetKubeConfigCmdExample = `$ otc-auth cce get-kube-config --cluster MyCluster --target-location /path/to/config

$ export CLUSTER_NAME=MyCluster
$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce get-kube-config --days-valid 14

$ export CLUSTER_NAME=MyCluster
$ export OS_DOMAIN_NAME=MyDomain
$ export OS_PROJECT_NAME=MyProject
$ otc-auth cce get-kube-config`

	//nolint:gosec // This is not a hardcoded credential but a help message containing "ak/sk"
	accessTokenCmdHelp = "Manage AK/SK"
	//nolint:gosec // This is not a hardcoded credential but a help message containing "ak/sk"
	accessTokenCreateCmdHelp = "Create new AK/SK"

	//nolint:gosec // This is not a hardcoded credential but a help message containing "ak/sk"
	accessTokenCreateCmdExample = `$ otc-auth access-token create --description "Custom token description"

$ otc-auth access-token create

$ export OS_DOMAIN_NAME=MyDomain
$ otc-auth access-token create`
	accessTokenListCmdHelp   = "List existing AK/SKs"
	accessTokenDeleteCmdHelp = "Delete existing AK/SKs"
	//nolint:gosec // This is not a hardcoded credential but a help message containing "ak/sk"
	accessTokenDeleteCmdExample = `$ otc-auth access-token delete --token YourToken

$ export OS_DOMAIN_NAME=YourDomain
$ export AK_SK_TOKEN=YourToken
$ otc-auth access-token delete

$ otc-auth access-token delete --token YourToken --os-domain-name YourDomain`
	//nolint:gosec // This example code does not actually contain credentials
	tempAccessTokenCreateCmdExample = `$ otc-auth temp-access-token create -t 900 -d YourDomainName # this creates a temp AK/SK which is 15 minutes valid (15 * 60 = 900)
	
	$ otc-auth temp-access-token create --duration-seconds 1800`
	openstackCmdHelp             = "Manage Openstack Integration"
	openstackConfigCreateCmdHelp = "Creates new clouds.yaml"
	usernameFlag                 = "os-username"
	skipTLSFlag                  = "skip-tls-verification"

	usernameShortFlag       = "u"
	skipTLSShortFlag        = ""
	usernameEnv             = "OS_USERNAME"
	usernameUsage           = "Username for the OTC IAM system. Either provide this argument or set the environment variable " + usernameEnv
	skipTLSUsage            = "Skip TLS Verification. This is insecure. Either provide this argument or set the environment variable " + skipTLSEnv
	passwordFlag            = "os-password"
	passwordShortFlag       = "p"
	passwordEnv             = "OS_PASSWORD"
	passwordUsage           = "Password for the OTC IAM system. Either provide this argument or set the environment variable " + passwordEnv
	domainNameFlag          = "os-domain-name"
	domainNameShortFlag     = "d"
	domainNameEnv           = "OS_DOMAIN_NAME"
	domainNameUsage         = "OTC domain name. Either provide this argument or set the environment variable " + domainNameEnv
	overwriteTokenFlag      = "overwrite-token"
	overwriteTokenShortFlag = "o"
	//nolint:gosec // This is not a hardcoded credential but a help message with a filename inside
	overwriteTokenUsage       = "Overrides .otc-info file"
	idpNameFlag               = "idp-name"
	idpNameShortFlag          = "i"
	idpNameEnv                = "IDP_NAME"
	idpNameUsage              = "Required for authentication with IdP"
	idpURLFlag                = "idp-url"
	idpURLEnv                 = "IDP_URL"
	idpURLUsage               = "Required for authentication with IdP"
	totpFlag                  = "totp"
	totpShortFlag             = "t"
	totpUsage                 = "6-digit time-based one-time password (TOTP) used for the MFA login flow. Needs to be used in conjunction with the " + userIDFlag + " flag or the " + userIDEnv + " environment variable"
	userIDFlag                = "os-user-domain-id"
	userIDEnv                 = "OS_USER_DOMAIN_ID"
	userIDUsage               = "User Id number, can be obtained on the \"My Credentials page\" on the OTC. Required if --totp is provided.  Either provide this argument or set the environment variable " + userIDEnv
	regionFlag                = "region"
	aliasFlag                 = "alias"
	aliasShortFlag            = "a"
	aliasUsage                = "Setting this changes the naming scheme for clusters in the Kube Config from {project name}/{cluster name} to the alias set"
	skipKubeTLSFlag           = "skip-kube-tls"
	skipKubeTLSUsage          = "Setting this adds the insecure-skip-tls-verify rule to the config for every cluster"
	regionShortFlag           = "r"
	regionEnv                 = "REGION"
	skipTLSEnv                = "SKIP_TLS_VERIFICATION"
	oidcScopesEnv             = "OIDC_SCOPES"
	oidcScopesFlag            = "oidc-scopes"
	oidcScopesShortFlag       = ""
	isServiceAccountFlag      = "service-account"
	isServiceAccountShortFlag = ""
	isServiceAccountUsage     = "Flag to be set when using a service account"
	oidcScopesUsage           = "Flag to set the scopes which are expected from the OIDC request. Either provide this argument or set the environment variable " + oidcScopesEnv

	clientIDEnv                                  = "CLIENT_ID"
	clientIDFlag                                 = "client-id"
	clientIDShortFlag                            = "c"
	clientIDUsage                                = "Client ID as set on the IdP. Either provide this argument or set the environment variable " + clientIDEnv
	clientSecretEnv                              = "CLIENT_SECRET"
	clientSecretFlag                             = "client-secret"
	clientSecretShortFlag                        = "s"
	clientSecretUsage                            = "Secret ID as set on the IdP. Either provide this argument or set the environment variable " + clientSecretEnv
	regionUsage                                  = "OTC region code. Either provide this argument or set the environment variable " + regionEnv
	projectNameFlag                              = "os-project-name"
	projectNameShortFlag                         = "p"
	projectNameEnv                               = "OS_PROJECT_NAME"
	projectNameUsage                             = "Name of the project you want to access. Either provide this argument or set the environment variable " + projectNameEnv
	printKubeConfigFlag                          = "output"
	printKubeConfigShortFlag                     = "o"
	printKubeConfigUsage                         = "Output fetched kube config to stdout instead of merging it with your existing kube config"
	printAkSkFlag                                = "output"
	printAkSkShortFlag                           = "o"
	printAkSkUsage                               = "Output contents of what would be written to ak-sk-env.sh to stdout instead"
	clusterNameFlag                              = "cluster"
	clusterNameShortFlag                         = "c"
	clusterNameEnv                               = "CLUSTER_NAME"
	clusterNameUsage                             = "Name of the clusterArg you want to access. Either provide this argument or set the environment variable " + clusterNameEnv
	daysValidFlag                                = "days-valid"
	daysValidDefaultValue                        = 7
	daysValidUsage                               = "Period (in days) that the config will be valid"
	serverFlag                                   = "server"
	serverShortFlag                              = "s"
	serverUsage                                  = "Override the server attribute in the kube config with the specified value"
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
	temporaryAccessTokenDurationSecondsUsage     = "The token's lifetime, in seconds. Valid times are between 900 and 86400 seconds"
	//nolint:gosec // This is not a hardcoded credential but a help message containing ak/sk
	accessTokenTokenUsage                        = "The AK/SK token to delete"
	openstackConfigCreateConfigLocationFlag      = "config-location"
	openstackConfigCreateConfigLocationShortFlag = "l"
	openstackConfigCreateConfigLocationUsage     = "Where the config should be saved"

	tempAccessTokenLifetime = 15 * 60 // 15 minutes
)
