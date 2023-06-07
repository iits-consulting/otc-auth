package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
)

func SetVersionInfo(cmd *cobra.Command, version, date string) {
	cmd.Version = fmt.Sprintf("%s built on %s", version, date)
}
