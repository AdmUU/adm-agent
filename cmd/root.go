/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/internal/processor"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/kardianos/service"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type program struct {
	configData config.Data
	shareData  config.Data
}

var (
    version   = "0.0.1"
    buildTime = "Unknown"
    gitCommit = "Unknown"
    goVersion = runtime.Version()
    platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	cmdVersion = false
	cmdVersionNumber = false
)

var log = utils.GetLogger()
var configData config.Data
var shareData  config.Data
var rootCmd = &cobra.Command{
	Use:   "adm-agent",
	Short: "Adm agent",
	Long:  "Adm agent",
	Args:  cobra.MinimumNArgs(0),
	PreRun: preRun,
	Run:   run,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configData.ConfigFile, "config", "c", "", "set configuration file")
	rootCmd.Flags().BoolVarP(&cmdVersion, "version", "V", false, "print version information")
	rootCmd.Flags().BoolVarP(&cmdVersionNumber, "", "v", false, "print version number")
}

func preRun(cmd *cobra.Command, args []string) {
	if cmdVersion {
		printVersion()
		os.Exit(0)
	}

	if cmdVersionNumber {
		printVersionNumber()
		os.Exit(0)
	}

	config.ReadConfig(configData.ConfigFile)
	utils.SetLoggerLevel()
	configData.ApiUrl = viper.GetString("api.url")
	shareData.ShareEnable = viper.GetString("share.enable")
	viper.Set("version", version)
}

func run(cmd *cobra.Command, args []string) {
	svcConfig := &service.Config{
		Name:        "adm-agent",
		DisplayName: "Adm agent",
		Description: "Adm agent",
		UserName:    "admuu",
        Option: service.KeyValue{
            "StartType": "automatic",
        },
		Dependencies: []string{
            "After=network.target",
            "After=nss-lookup.target",
        },
	}

	if configData.ConfigFile != "" {
		svcConfig.Arguments = append(svcConfig.Arguments, "--config", configData.ConfigFile)
	}

	prg := &program{
		configData: configData,
		shareData:  shareData,
	}

	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	if len(args) > 0 {
		action := args[0]
		switch action {
		case "status":
			status, err := s.Status()
			if err != nil {
				log.Fatal(err)
			}
			statusText := "unknown"
			switch status {
			case service.StatusRunning:
				statusText = "running"
			case service.StatusStopped:
				statusText = "stopped"
			}
			log.Infof("Service status: %s\n", statusText)
			return
		default:
			err := service.Control(s, action)
			if err != nil {
				log.Fatal(err)
			}
			return
		}
	}

	err = s.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func printVersion() {
    fmt.Print("Adm Agent <https://www.admin.im>\n\n")
    fmt.Printf("Version: \t%s\n", version)
    fmt.Printf("BuildTime: \t%s\n", buildTime)
    fmt.Printf("GitCommit: \t%s\n", gitCommit)
    fmt.Printf("GoVersion: \t%s\n", goVersion)
    fmt.Printf("Platform: \t%s\n", platform)
}

func printVersionNumber() {
	fmt.Println(version)
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	return nil
}

func (p *program) run() {
	ps := processor.Processor{ConfigData: p.configData, ShareData: p.shareData}
	if err := ps.Process(); err != nil {
		log.Fatal(err)
	}
}