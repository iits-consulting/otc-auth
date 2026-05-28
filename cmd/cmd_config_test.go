//nolint:testpackage // whitebox testing
package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestInitializeConfig_FlagSetFromEnvWhenMissing(t *testing.T) {
	t.Setenv(domainNameEnv, "MyDomain")
	t.Setenv(userIDEnv, "user-abc")

	var domainVal, userIDVal string
	cmd := &cobra.Command{Use: "test-cmd"}
	cmd.Flags().StringVar(&domainVal, domainNameFlag, "", "")
	cmd.Flags().StringVar(&userIDVal, userIDFlag, "", "")

	if err := initializeConfig(cmd, loginRemoveFlagToEnv); err != nil {
		t.Fatalf("initializeConfig returned error: %v", err)
	}

	if got, _ := cmd.Flags().GetString(domainNameFlag); got != "MyDomain" {
		t.Errorf("%s flag = %q, want %q (env var should populate flag)", domainNameFlag, got, "MyDomain")
	}
	if got, _ := cmd.Flags().GetString(userIDFlag); got != "user-abc" {
		t.Errorf("%s flag = %q, want %q (env var should populate flag)", userIDFlag, got, "user-abc")
	}
}

func TestInitializeConfig_ExplicitFlagBeatsEnv(t *testing.T) {
	t.Setenv(domainNameEnv, "FromEnv")

	var domainVal string
	cmd := &cobra.Command{Use: "test-cmd"}
	cmd.Flags().StringVar(&domainVal, domainNameFlag, "", "")
	if err := cmd.Flags().Set(domainNameFlag, "FromFlag"); err != nil {
		t.Fatalf("setting flag: %v", err)
	}

	if err := initializeConfig(cmd, loginRemoveFlagToEnv); err != nil {
		t.Fatalf("initializeConfig returned error: %v", err)
	}

	if got, _ := cmd.Flags().GetString(domainNameFlag); got != "FromFlag" {
		t.Errorf("%s flag = %q, want %q (explicit flag should take precedence)", domainNameFlag, got, "FromFlag")
	}
}

// TestFlagToEnvMaps_RequiredEntries would have caught the original bug where
// loginRemoveFlagToEnv was missing the OS_DOMAIN_NAME → --os-domain-name
// mapping (issue #177). It also guards every other command in the same way:
// any command that registers a flag listed in `requiredFlags` below MUST also
// wire its env counterpart in its flag→env map.
func TestFlagToEnvMaps_RequiredEntries(t *testing.T) {
	t.Parallel()
	cases := []struct {
		mapName       string
		flagToEnv     map[string]string
		requiredFlags map[string]string
	}{
		{
			mapName:   "loginIamFlagToEnv",
			flagToEnv: loginIamFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag: domainNameEnv,
				usernameFlag:   usernameEnv,
				passwordFlag:   passwordEnv,
				userIDFlag:     userIDEnv,
				regionFlag:     regionEnv,
			},
		},
		{
			mapName:   "loginIdpSamlFlagToEnv",
			flagToEnv: loginIdpSamlFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag: domainNameEnv,
				usernameFlag:   usernameEnv,
				passwordFlag:   passwordEnv,
				idpNameFlag:    idpNameEnv,
				idpURLFlag:     idpURLEnv,
				regionFlag:     regionEnv,
			},
		},
		{
			mapName:   "loginIdpOidcFlagToEnv",
			flagToEnv: loginIdpOidcFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag:   domainNameEnv,
				idpNameFlag:      idpNameEnv,
				idpURLFlag:       idpURLEnv,
				regionFlag:       regionEnv,
				clientIDFlag:     clientIDEnv,
				clientSecretFlag: clientSecretEnv,
				oidcScopesFlag:   oidcScopesEnv,
			},
		},
		{
			mapName:   "loginRemoveFlagToEnv",
			flagToEnv: loginRemoveFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag: domainNameEnv,
			},
		},
		{
			mapName:   "cceFlagToEnv",
			flagToEnv: cceFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag:  domainNameEnv,
				projectNameFlag: projectNameEnv,
			},
		},
		{
			mapName:   "cceListFlagToEnv",
			flagToEnv: cceListFlagToEnv,
			requiredFlags: map[string]string{
				regionFlag: regionEnv,
			},
		},
		{
			mapName:   "cceGetKubeConfigFlagToEnv",
			flagToEnv: cceGetKubeConfigFlagToEnv,
			requiredFlags: map[string]string{
				clusterNameFlag: clusterNameEnv,
				regionFlag:      regionEnv,
			},
		},
		{
			mapName:   "accessTokenFlagToEnv",
			flagToEnv: accessTokenFlagToEnv,
			requiredFlags: map[string]string{
				domainNameFlag: domainNameEnv,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.mapName, func(t *testing.T) {
			t.Parallel()
			for flag, wantEnv := range tc.requiredFlags {
				gotEnv, ok := tc.flagToEnv[flag]
				if !ok {
					t.Errorf("%s: missing required entry %q → %q (command cannot pick up env var)",
						tc.mapName, flag, wantEnv)
					continue
				}
				if gotEnv != wantEnv {
					t.Errorf("%s[%q] = %q, want %q", tc.mapName, flag, gotEnv, wantEnv)
				}
			}
		})
	}
}
