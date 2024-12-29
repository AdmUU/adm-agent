/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package socketio

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/admuu/adm-agent/pkg/components"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/gorilla/websocket"
)

func (s *SocketIO) handleEvent(event string, msg interface{}) error {
	var err error
	switch event {
	case "init":
		data := msg.(map[string]interface{})
		pInterval, err := data["pingInterval"].(float64)
		if !err {
			log.Error("PingInterval error")
		} else {
			s.keepPing(pInterval)
		}
		//s.keepAlive()
	case "connect":
		data := msg.(string)
		s.dialerTimes = 0
		log.Infof("Connection sid is %v\n", data)
	case "disconnect":
		s.conn.Close()
		log.Warn("Handle disconnect event.")
	case "disable":
		s.conn.Close()
		log.Warn("Handle disable event.")
	case "update":
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Warnf("Do for updates failed: %v ", r)
				}
			}()
			update := utils.Update{}
			if r := update.DoUpdate(); r != nil {
				log.Warnf("DoUpdate error: %v ", r)
			}

		}()
		log.Warn("Handle update event.")
	case "stop-task":
		taskId := msg.(string)
		if s.taskChanDone[taskId] != nil {
			close(s.taskChanDone[taskId])
		}
	case "block":
		log.Warn("Handle block event.")
		close(s.ConnectChanDone)
	case "err":
		log.Warnf("Received error message: %v\n", msg.(string))
	case "request-ping":
		data := msg.(map[string]interface{})
		if data["host"] == nil || data["pingtype"] == nil || data["protocol"] == nil || data["taskId"] == nil {
			return fmt.Errorf("event data format invalid")
		}
		host := strings.Trim(data["host"].(string), " \n\"'")
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = strings.Trim(host, "[]")
		}
		ip, host, port, ipVersion, err := utils.FilterIP(host)

		if err != nil {
			return fmt.Errorf("filterIP error: %v", err)
		}
		protocol := data["protocol"].(string)
		taskId := data["taskId"].(string)
		if protocol != "tcp" {
			port = ""
		}
		res := map[string]interface{}{
			"ip":        ip,
			"port":      port,
			"ipVersion": ipVersion,
			"taskType":  "ping",
			"taskId":    taskId,
		}
		s.sendMessage("agent-response", res)
		data["ip"] = ip
		data["host"] = host
		data["port"] = port
		data["ipVersion"] = ipVersion
		log.Debug(data)
		s.taskPing(data, taskId)
	case "agent-response":
		log.Debug("agent-response")
	default:
		log.Infof("Default event: [%v] %v\n", event, msg.(string))
	}
	return err
}

func (s *SocketIO) keepPing(pInterval float64) {
	pingInterval := time.Duration(pInterval) * time.Millisecond
	s.pingChanDone = make(chan struct{})

	go func(pingInterval time.Duration) {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("panic in keepPing: %v ", r)
			}
		}()
		for {
			select {
			case <-s.pingChanDone:
				return
			case <-ticker.C:
				if !s.heartbeatTime.IsZero() && time.Now().After(s.heartbeatTime.Add(20*time.Second)) {
					log.Errorf("Heartbeat timeout after %vs at %vs. Reconnect.", s.heartbeatTime, time.Now())
					s.conn.Close()
					close(s.pingChanDone)
					return
				}
				//s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(`2`)}
				s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(`2/agent`)}
			}
		}
	}(pingInterval)
}

func (s *SocketIO) keepAlive() {
	interval := 10 * time.Second
	eventName := "agent-keepalive"
	eventData := map[string]interface{}{
		"time": 0,
	}
	go func(interval time.Duration) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer func() {
			if r := recover(); r != nil {
				log.Warnf("panic in keepAlive: %v ", r)
			}
		}()
		for {
			<-ticker.C
			eventData["time"] = strconv.FormatInt(time.Now().Unix(), 10)
			message, _ := s.escapedString(eventName, eventData)
			s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(message)}
		}
	}(interval)
}

func (s *SocketIO) delay() {
	var (
		initialDelay = 2 * time.Second
		maxDelay     = 60 * time.Second
	)
	if s.dialerTimes == 0 {
		s.delayTime = time.Duration(0)
	} else if s.dialerTimes == 1 {
		s.delayTime = initialDelay
	} else if s.dialerTimes > 1 {
		s.delayTime *= 2
		if s.delayTime > maxDelay {
			s.delayTime = maxDelay
		}
	}
	time.Sleep(s.delayTime)
	s.dialerTimes++
}

func (s *SocketIO) sayHello() {
	eventName := "agent-task"
	eventData := map[string]interface{}{
		"token": s.token,
	}
	message, _ := s.escapedString(eventName, eventData)
	s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(message)}
}

func (s *SocketIO) taskPing(data map[string]interface{}, taskId string) {
	var pingCount = 3
	var loopCount = 1
	var delay float32
	var err error
	defer func() {
		if s.taskChanDone[taskId] != nil {
			close(s.taskChanDone[taskId])
			delete(s.taskChanDone, taskId)
		}
	}()
	ip := data["ip"].(string)
	protocol := data["protocol"].(string)
	pingtype := data["pingtype"].(string)
	if pingtype == "continuous" {
		pingCount = 1
		loopCount = 100
	}
	s.taskChanDone[taskId] = make(chan struct{})
	for i := 0; i < loopCount; i++ {
		select {
		case <-s.taskChanDone[taskId]:
			log.Infof("Task %v received stop signal...", taskId)
			return
		default:
			startTime := time.Now()
			if protocol == "icmp" {
				delay, err = components.IcmpPing(ip, pingCount)
			} else if protocol == "tcp" {
				delay, err = components.TcpPing(data)
			}
			if err != nil {
				log.WithError(err).Debug("ping error")
				delay = 0
			}
			res := map[string]interface{}{
				"delay":    delay,
				"taskType": "ping",
				"taskId":   data["taskId"].(string),
			}
			s.sendMessage("agent-response", res)
			duration := time.Since(startTime)
			if duration < 1*time.Second {
				time.Sleep(1 * time.Second)
			}
		}
	}
}
