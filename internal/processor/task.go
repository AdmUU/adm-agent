/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package processor

import (
	"fmt"
	"strings"

	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/adm"
	"github.com/admuu/adm-agent/pkg/socketio"
	"github.com/spf13/viper"
)

func getAuthCode(config *config.Data) error{
    var (
        err error
    )
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("panic occurred: %v", r)
        }
    }()

    nodeInfo, err := adm.RegistNode(config.ApiUrl, config.ApiKey, config.ApiSecret, *config)
    if err != nil {
        return err
    }

    config.ApiDid = nodeInfo.Did
    config.ApiAuthCode = nodeInfo.AuthCode
    config.ApiJar = nodeInfo.Jar
    if (config.ShareEnable == "yes") {
        viper.Set("share.enable", strings.ToLower(config.ShareEnable))
        viper.Set("share.name", config.ShareName)
        viper.Set("share.authcode", config.ApiAuthCode)
        viper.Set("share.did", config.ApiDid)
    } else {
        viper.Set("api.url", config.ApiUrl)
        viper.Set("api.authcode", config.ApiAuthCode)
        viper.Set("api.did", config.ApiDid)
    }

    err = viper.WriteConfig()
    if err != nil {
        return fmt.Errorf("error writing config file: %s", err)
    }
    return err
}

func socketConnect(apiUrl string, authcode string, config *config.Data) error {
    var err error
    defer func() {
        if r := recover(); r != nil {
            log.Errorf("panic occurred: %v", r)
        }
    }()

    io := socketio.SocketIO{ApiUrl: apiUrl, ApiAuthCode: authcode, ConfigData: config}
    if r := io.Run(); r != nil {
        err = fmt.Errorf("SocketIO error: %v", r)
    }
    return err
}
