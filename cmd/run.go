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
		managementAddress := ":33333"
		backendsFile := "./backends.yml"
		isVerbose := false
		username := ""
		password := ""
		timeout := 0
		whitelist := ""

		// Overrides
		for _, v := range args {
			argumentParts := strings.Split(v, "=")
			if len(argumentParts) == 2 {
				if argumentParts[0] == "address" {
					listenAddress = argumentParts[1]
				}
				if argumentParts[0] == "management" {
					managementAddress = argumentParts[1]
				}
				if argumentParts[0] == "backends" {
					backendsFile = argumentParts[1]
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
			}
		}

		s := models.Proxy{
			BackendsFile:  backendsFile,
			ListenAddress: listenAddress,
			Timeout:       timeout,
			Mutex:         &sync.Mutex{},
			SessionMutex:  &sync.Mutex{},
			Sessions:      map[string]string{},
			Username:      username,
			Password:      password,
			IsVerbose:     isVerbose,
			Whitelist:     whitelist,
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

		if managementAddress != "" {
			go models.Management{
				ListenAddress: managementAddress,
				Server:        &s,
			}.Listen()
		}

		s.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags()
}
