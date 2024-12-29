/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package socketio

import (
	"crypto/tls"
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/admuu/adm-agent/build/certs"
	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/adm"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/gorilla/websocket"
)

var err error
var log = utils.GetLogger()

type SocketIO struct {
	conn            *websocket.Conn
	messageChan     chan WebSocketMessage
	ConnectChanDone chan struct{}
	pingChanDone    chan struct{}
	taskChanDone    map[string]chan struct{}
	token           string
	delayTime       time.Duration
	heartbeatTime   time.Time
	dialerTimes     int
	ApiUrl          string
	ApiAuthCode     string
	ApiJar          *cookiejar.Jar
	ConfigData      *config.Data

}

type WebSocketMessage struct {
	messageType     int
	data            []byte
}

func (s *SocketIO) Run() error {
	s.dialerTimes = 0
	scheme, host, r := s.getSchemeHost()
	if r != nil {
		return fmt.Errorf("getSchemeHost error: %v", r)
	}

	s.ConnectChanDone = make(chan struct{})

	for {
		select {
		case <-s.ConnectChanDone:
			return fmt.Errorf("Connect to socket server %v blocked", host)
		default:
			if r := s.Connect(scheme, host); r != nil {
				log.WithError(r).Error("Run SocketIO error.")
				continue
			}
		}
	}
}

func (s *SocketIO) Connect(scheme string, host string) error {
	err = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in Connect: %v", r)
			log.Warn(err)
		}
	}()
	s.delay()
	
	var clientCert utils.Certificate
	if s.ConfigData.ShareEnable == "yes" {
        clientCert = utils.Certificate{
            CertPem: certs.GetCertPem(),
            CertKey: certs.GetCertKey(),
        }
    }
	
	tokenInfo, rcode, r  := adm.AgentTokenRequest(s.ApiUrl, s.ApiAuthCode, clientCert)
	if r != nil {
		if (rcode == 20015) {
			log.Warn("This node is blocked by the server.")
			close(s.ConnectChanDone)
		}
		return fmt.Errorf("GetToken failed: %v", r)
	}
	
	s.token = tokenInfo.Token
	u := url.URL{Scheme: scheme, Host: host, Path: "/socket.io/", RawQuery: "token=" + s.token + "&auth_code=" + s.ApiAuthCode}
	dialer := websocket.Dialer{
		Jar: tokenInfo.Jar,
	}
	
	if len(clientCert.CertPem) > 0 && len(clientCert.CertKey) > 0 {
		cert, err := tls.X509KeyPair(clientCert.CertPem, clientCert.CertKey)
		if err != nil {
			log.Fatalf("Error loading certificate and key: %v", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		dialer.TLSClientConfig = tlsConfig
	}
	
	s.conn, _, r = dialer.Dial(u.String(), nil)
	if r != nil {
		return fmt.Errorf("Connect to socket server %v failed: %v", host, r)
	}

	defer s.conn.Close()
	if s.ConfigData.ShareEnable != "yes" {
        log.Infof("Successfully connected to the socket server: %v", host)
    }
	s.messageChan = make(chan WebSocketMessage, 100)
	s.taskChanDone = make(map[string]chan struct{}, 100)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("panic in websocketWriter: %v ", r)
			}
		}()
		s.websocketWriter()
	}()

	s.sayHello()

	for {
		_, msg, r := s.conn.ReadMessage()
		if r != nil {
			log.WithError(r).Error("Read message error")
			break
		}
		packet, r := s.parseSocketMessage(msg)
		if r != nil {
			log.WithError(r).Debug("Failed to parse message")
			continue
		}
		if packet.Data == nil {
			continue
		}
		if packet.Event == "close" {
			log.Warn("Received a closed message")
			s.conn.Close()
			break
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("panic in handleEvent: %v", r)
				}
			}()
			r := s.handleEvent(packet.Event, packet.Data)
			if r != nil {
				log.WithError(r).Warn("handleEvent failed")
			}
		}()
		if (err != nil) {
			log.WithError(err).Warn("Failed to handle event")
		}
    }
	return err
}