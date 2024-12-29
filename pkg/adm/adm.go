/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package adm

import (
	"net/http/cookiejar"

	"github.com/admuu/adm-agent/build/certs"
	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/utils"
)

type NodeInfo struct {
    AuthCode string         `json:"auth_code"`
    Did      string         `json:"did"`
    Jar      *cookiejar.Jar `json:"-"`
}

type TokenInfo struct {
    Token    string         `json:"token"`
    Jar      *cookiejar.Jar `json:"-"`
}

func RegistNode(apiUrl string, key string, secret string, config config.Data) (*NodeInfo, error) {
    var clientCert utils.Certificate
	data, signature, err := makeRegistParam(key, secret, config)
	if err != nil {
        return nil,err
	}
    if config.ShareEnable == "yes" {
        clientCert = utils.Certificate{
            CertPem: certs.GetCertPem(),
            CertKey: certs.GetCertKey(),
        }
    }
    nodeInfo, err := registRequest(apiUrl, data, signature, clientCert)
	if err != nil {
        return nil,err
	}
    return nodeInfo,nil
}