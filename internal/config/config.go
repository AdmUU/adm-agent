/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package config

import (
	"net/http/cookiejar"
	"os"
	"path/filepath"

	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/spf13/viper"
)
var log = utils.GetLogger()
type Data struct {
	ApiUrl         string
	ApiKey         string
	ApiSecret      string
	ApiAuthCode    string
	ApiDid         string
	ApiJar         *cookiejar.Jar
	ConfigFile     string
	ShareEnable    string
	ShareName      string
	ShareSponsorID string
}
var Environment = "prod"
func ReadConfig(ConfigFile string) {
	if environment := os.Getenv("ADM_Environment"); environment != "" {
        Environment = environment
    }
	viper.SetDefault("api.url", "")
	viper.SetDefault("api.authcode", "")
	viper.SetDefault("api.did", "")
	viper.SetDefault("share.enable", "no")
	viper.SetDefault("share.name", "")
	viper.SetDefault("share.sponsorid", "")
	viper.SetDefault("share.authcode", "")
	viper.SetDefault("share.did", "")
	viper.SetDefault("app.env", Environment)
	viper.SetDefault("ip.prefer", "")
	
    if (ConfigFile != "") {
		viper.SetConfigFile(ConfigFile)
		log.Info("Load config file: " + ConfigFile)
	} else {
		configPath := ""
		// if (ConfigFile != "") {
		// 	viper.SetConfigFile(ConfigFile)
		// 	log.Debugf("Load config file: " + ConfigFile)
		// } else {
		// 	configPath := ""
		// 	configPath, _ = os.Getwd()
		// 	viper.AddConfigPath(configPath)
		// 	viper.SetConfigName("config")
		// 	viper.SetConfigType("yaml")
		// 	log.Debugf("No configuration file specified, load config file in " + configPath)
		// }
		if Environment == "dev" {
			configPath, _ = os.Getwd()
			viper.AddConfigPath(configPath)
		} else {
			ef, err := os.Executable()
			if err != nil {
				panic(err)
			}
			configPath = filepath.Dir(ef)
			viper.AddConfigPath(configPath)
			log.Debugf("Load config file in " + configPath)
		}
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	err := viper.ReadInConfig()
	if err != nil {
		if (ConfigFile != "") {
			err = viper.SafeWriteConfigAs(ConfigFile)
		} else {
			err = viper.SafeWriteConfig()
		}
		if err != nil {
			panic(err)
		}
	}
}