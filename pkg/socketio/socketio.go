/*
Copyright © 2024 Admin.IM <dev@admin.im>
*/
package socketio

import (
	"fmt"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/admuu/adm-agent/build/certs"
	"github.com/admuu/adm-agent/internal/config"
	"github.com/admuu/adm-agent/pkg/adm"
	"github.com/admuu/adm-agent/pkg/network"
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
				log.Errorf("Run SocketIO error: %v", r)
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

	var clientCert *network.Certificate
	if s.ConfigData.ShareEnable == "yes" {
        clientCert = &network.Certificate{
            CertPem: certs.GetCertPem(),
            CertKey: certs.GetCertKey(),
        }
    }

	tokenInfo, rcode, r  := adm.AgentTokenRequest(s.ApiUrl, s.ApiAuthCode, s.ConfigData.ApiSecret, clientCert)
	if r != nil {
		if (rcode == 20015) {
			log.Warn("This node is blocked by the server.")
			close(s.ConnectChanDone)
		}
		return fmt.Errorf("GetToken failed: %v", r)
	}
	var reqSign string
	urlPath := "/socket.io/"
	if clientCert != nil {
		reqSign = "&reqsign=" + adm.GenerateReqSign(urlPath, s.ConfigData.ApiSecret)
	}
	s.token = tokenInfo.Token
	rq := fmt.Sprintf("token=%s&auth_code=%s%s",
	s.token,
	s.ApiAuthCode,
	reqSign)
	u := url.URL{Scheme: scheme, Host: host, Path: urlPath, RawQuery: rq}
	ws := network.Websocket{Url: u.String(), Jar: tokenInfo.Jar, Certificate: clientCert}
	s.conn, _, r = ws.Dial()
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
			log.Errorf("Read message error: %v", r)
			break
		}
		packet, r := s.parseSocketMessage(msg)
		if r != nil {
			log.Debugf("Failed to parse message: %v", r)
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
				log.Warnf("handleEvent failed: %v", r)
			}
		}()
		if (err != nil) {
			log.Warnf("Failed to handle event: %v", err)
		}
    }
	return err
}