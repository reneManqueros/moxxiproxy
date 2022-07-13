package cmd

import (
	"github.com/spf13/cobra"
	"moxxiproxy/models"
	"strconv"
	"strings"
	"sync"
	"time"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run proxy server",
	Run: func(cmd *cobra.Command, args []string) {
		// Defaults
		listenAddress := "0.0.0.0:1989"
		isVerbose := false
		timeout := 0
		whitelist := ""
		isUpstream := false
		users := "users.yml"
		exitNodes := "exitNodes.yml"
		hideDown := false

		// Overrides
		for _, v := range args {
			argumentParts := strings.Split(v, "=")
			if len(argumentParts) == 2 {
				if argumentParts[0] == "address" {
					listenAddress = argumentParts[1]
				}
				if argumentParts[0] == "timeout" {
					timeout, _ = strconv.Atoi(argumentParts[1])
				}
				if argumentParts[0] == "users" {
					users = argumentParts[1]
				}
				if argumentParts[0] == "exitNodes" {
					exitNodes = argumentParts[1]
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
				if argumentParts[0] == "hidedown" && argumentParts[1] == "true" {
					hideDown = true
				}
			}
		}
		models.ServerHealth.Init()
		if hideDown == true {
			go func() {
				for {
					time.Sleep(1801 * time.Second)
					models.ServerHealth.ReviveOld()
				}
			}()
		}

		s := models.ProxyServer{
			ConfigFiles: struct {
				Users string
				Nodes string
			}{
				Users: users,
				Nodes: exitNodes,
			},
			ExitNodes: struct {
				All          []models.ExitNode
				ByRegion     map[string][]models.ExitNode
				ByInstanceID map[string]models.ExitNode
			}{
				All:          []models.ExitNode{},
				ByRegion:     map[string][]models.ExitNode{},
				ByInstanceID: map[string]models.ExitNode{},
			},
			IsUpstream:    isUpstream,
			IsVerbose:     isVerbose,
			HideDown:      hideDown,
			ListenAddress: listenAddress,
			Mutex:         &sync.Mutex{},
			Sessions:      map[string]models.ExitNode{},
			SessionMutex:  &sync.Mutex{},
			Timeout:       timeout,
			Whitelist:     whitelist,
		}

		s.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.Flags()
}
