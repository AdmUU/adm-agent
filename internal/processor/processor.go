/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package processor

import (
	"os"
	"sync"

	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/spf13/viper"
)

type Processor struct {
	ConfigData config.Data
	ShareData  config.Data
}

var (
    wg            sync.WaitGroup
    ShareUrl      string
    ShareKey      string
    ShareSecret   string
)

var log = utils.GetLogger()

func (ps *Processor) preRun() {
	if (viper.GetString("app.env") != "dev") {
		return
	}
    if envURL := os.Getenv("ADM_SHARE_URL"); envURL != "" {
        ShareUrl = envURL
    }
    if envKey := os.Getenv("ADM_SHARE_KEY"); envKey != "" {
        ShareKey = envKey
    }
    if envSecret := os.Getenv("ADM_SHARE_SECRET"); envSecret != "" {
        ShareSecret = envSecret
    }
}

func (ps *Processor) Process() error {
	var err error
	var isProcess bool
    defer func() {
        if r := recover(); r != nil {
			log.Errorf("panic occurred: %v", r)
        }
    }()

	ps.preRun()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("Check for updates failed: %v ", r)
			}
		}()
		update := utils.Update{}
		if r := update.CheckUpdate(); r != nil {
			log.Warnf("CheckUpdate error: %v ", r)
		}
	}()

    if (ps.ConfigData.ApiUrl != "" && viper.GetString("api.authcode") != "") {
		isProcess = true
		wg.Add(1)
        go func() {
            defer wg.Done()
            if err := socketConnect(ps.ConfigData.ApiUrl, viper.GetString("api.authcode"), &ps.ConfigData); err != nil {
				log.Errorf("SocketConnect failed: %v", err)
            }
        }()
	}
	if (ps.ShareData.ShareEnable == "yes" && ShareUrl != "") {
		ps.ShareData.ApiUrl = ShareUrl
		ps.ShareData.ApiKey = ShareKey
		ps.ShareData.ApiSecret = ShareSecret
		log.Info("Share server enable")

		if (viper.GetString("share.authcode") != "") {
			isProcess = true
			wg.Add(1)
            go func() {
                defer wg.Done()
                if err := socketConnect(ps.ShareData.ApiUrl, viper.GetString("share.authcode"), &ps.ShareData); err != nil {
					log.Errorf("SocketConnect share server failed: %v", err)
                }
            }()
		}
	}
	wg.Wait()
	if !isProcess {
		log.Fatal("No valid configuration.")
	}
	return err
}

func (ps *Processor) Register() {
	var isProcess bool
    defer func() {
        if r := recover(); r != nil {
			log.Errorf("panic occurred: %v", r)
        }
    }()

	ps.preRun()

	if ps.ConfigData.ApiUrl != "" && ps.ConfigData.ApiKey != "" && ps.ConfigData.ApiSecret != "" {
		if err := getAuthCode(&ps.ConfigData); err != nil {
			log.Fatal("Get authCode failed: " + err.Error())
		}
		isProcess = true
	}

	log.Info("Share enable: "+ps.ShareData.ShareEnable)
	if (ps.ShareData.ShareEnable == "yes" && ShareUrl != "") {
		log.Debug("ShareUrl: " + ShareUrl)
		ps.ShareData.ApiUrl = ShareUrl
		if (ShareKey != "" && ShareSecret != "") {
			ps.ShareData.ApiKey = ShareKey
			ps.ShareData.ApiSecret = ShareSecret
			if err := getAuthCode(&ps.ShareData); err != nil {
				log.Fatal("Get share authCode failed: " + err.Error())
			}
			isProcess = true
		}
	}

	if isProcess {
		log.Info("Successful registration.")
	} else {
		log.Fatal("Invalid registration data.")
	}
}
