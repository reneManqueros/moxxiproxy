package cmd

import (
	"github.com/spf13/cobra"
	"moxxiproxy/models"
	"strings"
	"sync"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		listenAddress, _ := cmd.Flags().GetString("address")
		exitnodesFile, _ := cmd.Flags().GetString("exitnodes")
		whitelist, _ := cmd.Flags().GetString("whitelist")
		auth, _ := cmd.Flags().GetString("auth")
		timeout, _ := cmd.Flags().GetInt("timeout")
		isVerbose, _ := cmd.Flags().GetBool("verbose")
		isUpstream, _ := cmd.Flags().GetBool("upstream")

		username := ""
		password := ""
		if authParts := strings.Split(auth, ":"); len(authParts) > 1 {
			username = authParts[0]
			password = authParts[1]
		}

		s := models.Proxy{
			ExitNodesFile: exitnodesFile,
			ListenAddress: listenAddress,
			Timeout:       timeout,
			Mutex:         &sync.Mutex{},
			SessionMutex:  &sync.Mutex{},
			Sessions:      map[string]string{},
			Username:      username,
			Password:      password,
			IsVerbose:     isVerbose,
			Whitelist:     whitelist,
			IsUpstream:    isUpstream,
			ExitNodes: struct {
				All          []models.ExitNode
				ByRegion     map[string][]models.ExitNode
				ByInstanceID map[string]models.ExitNode
			}{
				All:          []models.ExitNode{},
				ByRegion:     map[string][]models.ExitNode{},
				ByInstanceID: map[string]models.ExitNode{},
			},
		}
		s.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().String("address", "0.0.0.0:1989", "--address=:1989")
	runCmd.PersistentFlags().String("exitnodes", "./exitNodes.yml", "--exitnodes=./exitnodes.yml")
	runCmd.PersistentFlags().String("auth", "", "--auth=user:pass")
	runCmd.PersistentFlags().String("whitelist", "", "--whitelist=1.2.3.4,5.6.7.8")
	runCmd.PersistentFlags().Int("timeout", 0, "--timeout=0")
	runCmd.PersistentFlags().Bool("verbose", false, "--verbose=false")
	runCmd.PersistentFlags().Bool("upstream", false, "--upstream=false")
	runCmd.Flags()
}
