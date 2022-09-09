package cmd

import (
	"github.com/spf13/cobra"
	"moxxiproxy/models"
	"strconv"
	"strings"
	"sync"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		// Defaults
		listenAddress := "0.0.0.0:1989"
		exitnodesFile := "./exitnodes.yml"
		isVerbose := false
		username := ""
		password := ""
		timeout := 0
		whitelist := ""
		isUpstream := false
		// Overrides
		for _, v := range args {
			argumentParts := strings.Split(v, "=")
			if len(argumentParts) == 2 {
				if argumentParts[0] == "address" {
					listenAddress = argumentParts[1]
				}

				if argumentParts[0] == "exitnodes" {
					exitnodesFile = argumentParts[1]
				}
				if argumentParts[0] == "timeout" {
					timeout, _ = strconv.Atoi(argumentParts[1])
				}
				if argumentParts[0] == "auth" {
					if authParts := strings.Split(argumentParts[1], ":"); len(authParts) > 1 {
						username = authParts[0]
						password = authParts[1]
					}
				}
				if argumentParts[0] == "whitelist" {
					whitelist = argumentParts[1]
				}
				if argumentParts[0] == "verbose" && argumentParts[1] == "true" {
					isVerbose = true
				}
				if argumentParts[0] == "upstream" && argumentParts[1] == "true" {
					isUpstream = true
				}
			}
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
	runCmd.Flags()
}
