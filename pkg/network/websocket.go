/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/

package network

import (
	"crypto/tls"
	_ "embed"
	"net"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/gorilla/websocket"
	"github.com/spf13/viper"
)

type Websocket struct {
	Url         string
	Timeout     time.Duration
	Jar         *cookiejar.Jar
	Certificate *Certificate
}

func (w *Websocket) Dial() (*websocket.Conn, *http.Response, error)  {
	if w.Timeout == 0 {
		w.Timeout = 30
	}

	netdialer := &NetDialer{
        Timeout:       10 * time.Second,
        KeepAlive:    60 * time.Second,
        fallbackDelay: 300 * time.Millisecond,
        resolver:     net.DefaultResolver,
    }

	wd := &websocket.Dialer{
        HandshakeTimeout: w.Timeout * time.Second,
        NetDialContext:   netdialer.DialContext,
		Jar: w.Jar,
    }

	if w.Certificate != nil {
		cert, err := tls.X509KeyPair(w.Certificate.CertPem, w.Certificate.CertKey)
		if err != nil {
			log.Fatalf("Error loading certificate and key: %v", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion: tls.VersionTLS12,
		}
		wd.TLSClientConfig = tlsConfig

	}

	header := http.Header{}
    header.Set("User-Agent", "Adm-agent/" + viper.GetString("version"))

	return wd.Dial(w.Url, header)
}
