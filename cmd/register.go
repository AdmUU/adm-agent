/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package cmd

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/internal/processor"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a node",
	Long:  `Register a node to get an authorization code`,
	PreRun: registerPreRun,
	Run: func(cmd *cobra.Command, args []string) {
		ps := processor.Processor{ConfigData: configData, ShareData: shareData}
		ps.Register()
	},
}

func init() {
	rootCmd.AddCommand(registerCmd)
	registerCmd.Flags().StringVarP(&configData.ApiUrl, "api", "a", "", "Server api url")
	registerCmd.Flags().StringVarP(&configData.ApiKey, "key", "k", "", "Api key")
	registerCmd.Flags().StringVarP(&configData.ApiSecret, "secret", "s", "", "Api secret")
	registerCmd.Flags().StringVarP(&shareData.ShareEnable, "share", "", "", "Share node (yes|no)")
	registerCmd.Flags().StringVarP(&shareData.ShareName, "sharename", "", "", "Your share name")
	registerCmd.Flags().StringVarP(&shareData.ShareSponsorID, "sponsorid", "", "", "Your share id")
}

func registerPreRun(cmd *cobra.Command, args []string) {
	config.ReadConfig(configData.ConfigFile)
	utils.SetLoggerLevel()

	if configData.ApiUrl != "" {
		_, err := url.ParseRequestURI(configData.ApiUrl)
		if err != nil || (configData.ApiUrl[:7] != "http://" && configData.ApiUrl[:8] != "https://") {
			log.Fatal("The API address is invalid")
		}
	} else {
		configData.ApiUrl = viper.GetString("api.url")
	}

	if configData.ApiKey != "" {
		keyRegex := regexp.MustCompile(`^[a-zA-Z0-9]{8}$`)
		if !keyRegex.MatchString(configData.ApiKey) {
			log.Fatal("Key must be 8 alphanumeric characters")
		}
	}

	if configData.ApiSecret != "" {
		secretRegex := regexp.MustCompile(`^[a-zA-Z0-9]{16}$`)
		if !secretRegex.MatchString(configData.ApiSecret) {
			log.Fatal("Secret must be 16 alphanumeric characters")
		}
	}

	if shareData.ShareEnable != "" {
		if  !strings.EqualFold(shareData.ShareEnable, "yes") && !strings.EqualFold(shareData.ShareEnable, "no") {
			log.Fatal("Share must be yes or no")
		}
		shareData.ShareEnable = strings.ToLower(shareData.ShareEnable)
	} else if viper.GetString("share.enable") != "" {
		shareData.ShareEnable = strings.ToLower(viper.GetString("share.enable"))
	} else {
		shareData.ShareEnable = "no"
	}

	if shareData.ShareEnable == "yes" && shareData.ShareName != "" {
		nameRegex := regexp.MustCompile(`^[\p{Han}a-zA-Z][\p{Han}a-zA-Z0-9_-]{1,19}$`)
		if !nameRegex.MatchString(shareData.ShareName) {
			log.Fatal("Share name must be 2 to 20 characters, start with a letter, and can only contain letters, numbers, underscores, and hyphens")
		}
	} else {
		shareData.ShareName = viper.GetString("share.name")
	}

	if configData.ApiUrl == "" && shareData.ShareEnable == "no" {
		log.Fatal("Please specify your server address")
	}

	configData.ApiAuthCode = viper.GetString("api.authcode")
	shareData.ApiAuthCode = viper.GetString("share.authcode")
	shareData.ShareSponsorID = viper.GetString("api.sponsorid")
	viper.Set("version", version)
}
