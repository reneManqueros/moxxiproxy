package cmd

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"moxxiproxy/models"
	"os"
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
		loglevel, _ := cmd.Flags().GetString("loglevel")
		timeout, _ := cmd.Flags().GetInt("timeout")
		isUpstream, _ := cmd.Flags().GetBool("upstream")
		prettyLogs, _ := cmd.Flags().GetBool("prettylogs")
		metricsLogger, _ := cmd.Flags().GetString("metrics")
		promaddress, _ := cmd.Flags().GetString("promaddress")
		if prettyLogs == true {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		}

		if metricsLogger != "" && metricsLogger != "prometheus" && metricsLogger != "stdout" {
			log.Fatal().Msg("Invalid metrics logger")
		}

		if logLevel, err := zerolog.ParseLevel(loglevel); err == nil {
			if logLevel == zerolog.NoLevel {
				zerolog.SetGlobalLevel(zerolog.InfoLevel)
			} else {
				zerolog.SetGlobalLevel(logLevel)
			}
		}

		username := ""
		password := ""
		if authParts := strings.Split(auth, ":"); len(authParts) > 1 {
			username = authParts[0]
			password = authParts[1]
			models.UserMap = make(map[string]models.User)
			models.UserMap[username] = models.User{
				UserID:    username,
				AuthToken: password,
			}
		}

		s := models.Proxy{
			ExitNodesFile: exitnodesFile,
			ListenAddress: listenAddress,
			Timeout:       timeout,
			Mutex:         &sync.Mutex{},
			SessionMutex:  &sync.Mutex{},
			Sessions:      map[string]models.ExitNode{},
			Username:      username,
			Password:      password,
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
			LogMetrics:        metricsLogger != "",
			MetricsLogger:     metricsLogger,
			PrometheusAddress: promaddress,
		}
		s.Run()
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
	runCmd.PersistentFlags().String("metrics", "", "--metrics=prometheus,stdout or --metrics=stdout")
	runCmd.PersistentFlags().String("promaddress", "0.0.0.0:2122", "--promaddress=:2122")
	runCmd.PersistentFlags().String("address", "0.0.0.0:1989", "--address=:1989")
	runCmd.PersistentFlags().String("exitnodes", "./exitNodes.yml", "--exitnodes=./exitnodes.yml")
	runCmd.PersistentFlags().String("auth", "", "--auth=user:pass")
	runCmd.PersistentFlags().String("whitelist", "", "--whitelist=1.2.3.4,5.6.7.8")
	runCmd.PersistentFlags().String("loglevel", "info", "--loglevel=info")
	runCmd.PersistentFlags().Int("timeout", 0, "--timeout=0")
	runCmd.PersistentFlags().Bool("upstream", false, "--upstream=false")
	runCmd.PersistentFlags().Bool("prettylogs", false, "--prettylogs=true")
	runCmd.PersistentFlags().Bool("verbose", false, "DEPRECATED, use loglevel instead")
	runCmd.Flags()
}
