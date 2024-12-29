/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package adm

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/spf13/viper"
)

func makeRegistParam(key string, secret string, config config.Data) (string, string, error)  {
	fingerprint := utils.GenerateFingerprint()
    timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceArr := make([]byte, 16)
    rand.Read(nonceArr)
	nonce := hex.EncodeToString(nonceArr)
    params := map[string]string{
        "key": key,
        "fingerprint":   fingerprint,
        "timestamp":   timestamp,
        "nonce":   nonce,
    }

    if (config.ApiAuthCode != "") {
        params["auth_code"] = config.ApiAuthCode
    }
    if (config.ShareName != "") {
        params["sponsor"] = config.ShareName
    }
    if (config.ShareSponsorID != "") {
        params["sponsor_id"] = config.ShareSponsorID
    }

    ipv4, err := utils.GetIP("ipv4")
    if ipv4 != nil && err == nil {
        params["ipv4"] = ipv4.(string)
    }

    ipv6, err := utils.GetIP("ipv6")
    if ipv6 != nil && err == nil {
        params["ipv6"] = ipv6.(string)
    }

    if ipv4 == nil && ipv6 != nil {
        viper.Set("ip.prefer", "ipv6")
    }

    if ipv4 == nil && ipv6 == nil {
        return "", "", errors.New("unable to obtain the IP address of this server")
    }

    params["version"] = viper.GetString("version")

    log.Debug("Regist params:", params)

    keys := make([]string, 0, len(params))
    for k := range params {
        keys = append(keys, k)
    }
    sort.Strings(keys)

	message := ""
    formValues := url.Values{}
    for _, k := range keys {
		message += k + "=" + url.QueryEscape(params[k]) + "&"
        formValues.Set(k, params[k])
    }
	message = message[:len(message)-1]

    postBody := formValues.Encode()
    hmacSha256 := hmac.New(sha256.New, []byte(secret))
    hmacSha256.Write([]byte(message))
    signature := hex.EncodeToString(hmacSha256.Sum(nil))
    return postBody, signature, nil
}