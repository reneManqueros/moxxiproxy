package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"moxxiproxy/models"
	"strings"
)

var removeCmd = &cobra.Command{
	Use:   "remove <backend>",
	Short: "Remove a backend from the pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		backend := strings.ToLower(args[0])
		message := fmt.Sprintf(`-%v`, backend)

		managementAddress := ":33333"
		for _, v := range args {
			argumentParts := strings.Split(v, "=")
			if len(argumentParts) == 2 {
				if argumentParts[0] == "management" {
					managementAddress = argumentParts[1]
				}
			}
		}

		models.Management{
			ListenAddress: managementAddress,
		}.Send(message)

		log.Println("backend removed", backend)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
	removeCmd.Flags()
}
