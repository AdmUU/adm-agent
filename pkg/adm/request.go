/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package adm

import (
	"fmt"

	"github.com/admuu/adm-agent/pkg/network"
	"github.com/admuu/adm-agent/pkg/utils"
)

var log = utils.GetLogger()

func registRequest(apiUrl string, requestData string, signature string, secret string, clientCert *network.Certificate) (*NodeInfo, error)  {
    var nodeInfo NodeInfo
    var reqSign string
    urlPath := "/api/adm/v1/registNode"
    if clientCert != nil {
        reqSign = "&reqsign=" + GenerateReqSign(urlPath, secret)
    }

    log.Debug("Register a node on ", apiUrl)
    url := fmt.Sprintf("%s%s?signature=%s%s",
        apiUrl,
        urlPath,
        signature,
        reqSign)
    http := network.Http{Url: url, Method: "POST", Data: requestData, Certificate: clientCert}
    response, err := http.ApiRequest()
    if err != nil {
        return nil,err
    }

    switch data := response.Data.(type) {
    case map[string]interface{}:
        if _, exists := data["auth_code"]; exists {
            nodeInfo.AuthCode = data["auth_code"].(string)
        }
        if _, exists := data["did"]; exists {
            nodeInfo.Did = data["did"].(string)
        }
    default:
        return nil,fmt.Errorf("registNode response data is of unexpected type")
    }
    nodeInfo.Jar = response.Jar
    return &nodeInfo,nil
}

func AgentTokenRequest(apiUrl string, authCode string, secret string, clientCert *network.Certificate) (*TokenInfo, int, error)  {
    var tokenInfo TokenInfo
    var reqSign string
    urlPath := "/api/adm/v1/requestAgentToken"
    if clientCert != nil {
        reqSign = GenerateReqSign(urlPath, secret)
    }
    url := fmt.Sprintf("%s%s?auth_code=%s&reqsign=%s",
        apiUrl,
        urlPath,
        authCode,
        reqSign)
    http := network.Http{Url: url, Method: "POST", Certificate: clientCert}
    response, err := http.ApiRequest()
    if err != nil {
        return nil, response.Code, err
    }
    switch data := response.Data.(type) {
    case map[string]interface{}:
        if _, exists := data["token"]; exists {
            tokenInfo.Token = data["token"].(string)
        }
    default:
        return nil, response.Code, fmt.Errorf("requestAgentToken response data is of unexpected type")
    }
    tokenInfo.Jar = response.Jar
    return &tokenInfo, response.Code, nil
}