package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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

func configureCmdFlagsAgainstEnvs(flagToEnvMapping map[string]string) func(*cobra.Command, []string) error {
	//nolint:revive // args is used later
	return func(cmd *cobra.Command, args []string) error {
		return initializeConfig(cmd, flagToEnvMapping)
	}
}
