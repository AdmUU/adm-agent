/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package utils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/spf13/viper"
)

type Update struct {
	UpdateProvider   string
	latest           *selfupdate.Release
	updater          *selfupdate.Updater
	need             bool
}

var (
    ReleaseUrl string
)

func (u *Update) preRun() {
	if (viper.GetString("app.env") != "dev") {
		return
	}
    if envReleaseUrl := os.Getenv("ADM_RELEASE_URL"); envReleaseUrl != "" {
        ReleaseUrl = envReleaseUrl
    }
}

func (u *Update) CheckUpdate() error {
	var source selfupdate.Source
	var err error
	var found bool
	var header http.Header
	u.need = false

	u.preRun()

	if (u.UpdateProvider == "") {
		u.UpdateProvider = "http"
	}

	switch u.UpdateProvider {
	case "http":
		if ReleaseUrl == "" {
			return errors.New("ReleaseUrl is empty")
		}
		header = make(http.Header)
		header.Set("User-Agent", "Adm-agent/" + viper.GetString("version"))
		source, err = selfupdate.NewHttpSource(selfupdate.HttpConfig{
			BaseURL: ReleaseUrl,
			Headers: header,
		})
	default:
		source, err = selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	}
	if err != nil {
		return err
	}
	u.updater, err = selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})

	if err != nil {
        return err
	}
	u.latest, found, err = u.updater.DetectLatest(context.Background(), selfupdate.ParseSlug("admuu/adm-agent"))
	if err != nil {
		return fmt.Errorf("error occurred while detecting version: %w", err)
	}
	if !found {
		return fmt.Errorf("latest version for %s/%s could not be found from github repository", runtime.GOOS, runtime.GOARCH)
	}

	version := viper.GetString("version")
	if u.latest.LessOrEqual(version) {
		log.Infof("Current version (%s) is the latest", version)
		return nil
	}
	if u.latest.GreaterThan(version) {
		u.need = true
		log.Infof("New version %s is available", u.latest.Version())
		return nil
	}
	return fmt.Errorf("unable to compare current version %v with latest version %v", version, u.latest.Version())


}

func (u *Update) DoUpdate() error {
	err := u.CheckUpdate()
	if err != nil {
		return err
	}
	if !u.need {
		return nil
	}
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return errors.New("could not locate executable path")
	}
	if err := u.updater.UpdateTo(context.Background(), u.latest, exe); err != nil {
		return fmt.Errorf("error occurred while updating binary: %w", err)
	}
	log.Infof("Successfully updated to version %s", u.latest.Version())
	os.Exit(1)
	return nil
}