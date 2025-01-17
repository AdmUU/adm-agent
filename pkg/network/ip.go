/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package network

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/spf13/viper"
)

type myIP struct {
	IP interface{}        `json:"ip"`
}

func FilterIP(input string, args ...string) (ip string, host string, port string, ipVersion string, err error) {
	//prefer := "ipv4"
	var prefer string
	if len(args) > 0 {
		prefer = args[0]
	} else if viper.IsSet("ip.prefer") && viper.GetString("ip.prefer") != "" {
		prefer = viper.GetString("ip.prefer")
	}

	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		u, err := url.Parse(input)
		if err != nil {
			return "", "", "", "", fmt.Errorf("invalid url: %s", input)
		}
		host = u.Hostname()
		port = u.Port()
		if port == "" {
			if u.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}
	} else {
		host, port, err = net.SplitHostPort(input)
		if err != nil {
			host = input
			port = "80"
		}
	}

	ip, host, ipVersion, err = parseDomainIP(host, prefer)
	return ip, host, port, ipVersion, err
}


func GetIP(ipVersion string) (interface{}, error)  {
    var ip string
    var respData myIP
    var ipApi string
    var tcpType string

	switch ipVersion {
	case "ipv4":
		ipApi = "ipv4"
		tcpType = "tcp4"
	case "ipv6":
		ipApi = "ipv6"
		tcpType = "tcp6"
	default:
		ipApi = "ip"
		tcpType = "tcp"
	}
    ipApiUrl := "https://" + ipApi + ".001000.best"
    http := Http{Url: ipApiUrl, Method: "GET", Data: map[string]string{"format":"json"}, NetworkType: tcpType, Timeout: 10}
    response, err := http.UrlRequest()
    if err != nil {
        return nil, err
    }
	err = json.Unmarshal([]byte(response), &respData)
	if err != nil {
		return response, err
	}
	if respData.IP != nil {
		ip = respData.IP.(string)
	}
	return ip, nil
}

func isDomainRegex(domain string) bool {
	pattern := `^(?:[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$`
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(strings.ToLower(domain))
}

func domainToIP(domain string, prefer string) (net.IP, error) {
	ips, err := net.LookupIP(domain)
	if err != nil {
		return nil, err
	}
	if prefer != "" {
		for _, ip := range ips {
			if prefer == "ipv4" && net.ParseIP(ip.String()).To4() != nil {
				return ip, nil
			} else if prefer == "ipv6" && net.ParseIP(ip.String()).To4() == nil {
				return ip, nil
			}
		}
	}
	return ips[0], nil
}

func parseDomainIP(input string, prefer string) (ip string, host string, ipVersion string, err error) {
	host = input
	ipAddr := net.ParseIP(host)
	if ipAddr == nil {
		if isDomainRegex(host) {
			ipAddr, err = domainToIP(host, prefer)
			if err != nil {
				return "", "", "", fmt.Errorf("invalid domain: %s", host)
			}
		} else {
			return "", "", "", fmt.Errorf("invalid IP address: %s", host)
		}
	}

	if ipAddr.To4() != nil {
		ipVersion = "IPv4"
	} else {
		ipVersion = "IPv6"
	}
	return ipAddr.String(), host, ipVersion, nil
}
