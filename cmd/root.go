package cmd

import "github.com/spf13/cobra"

var RootCmd = &cobra.Command{
	Use:   "miner-pool",
	Short: "Open source, miner pool",
}

func init() {
	RootCmd.PersistentFlags().StringP("config", "c", "", "Configuration file to use.")
}

func Run(args []string) error {
	RootCmd.SetArgs(args)
	return RootCmd.Execute()
}
