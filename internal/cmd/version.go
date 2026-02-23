package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/joern1811/wachat/internal/version"
)

var versionLong bool

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		info := version.Get()
		if versionLong {
			out, _ := json.MarshalIndent(info, "", "  ")
			fmt.Println(string(out))
		} else {
			fmt.Println(info.Version)
		}
	},
}

func init() {
	versionCmd.Flags().BoolVar(&versionLong, "long", false, "Print detailed version information as JSON")
	rootCmd.AddCommand(versionCmd)
}
