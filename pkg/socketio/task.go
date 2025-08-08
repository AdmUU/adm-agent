// Copyright 2024-2025 Admin.IM <dev@admin.im>
// SPDX-License-Identifier: GPL-3.0-or-later

package socketio

import (
	"fmt"
	"strconv"
	"time"

	"github.com/admuu/adm-agent/pkg/components"
	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/gorilla/websocket"
)

// initTaskHandlers initializes task registry and registers task handlers
func (s *SocketIO) initTaskHandlers() {
	s.taskRegistry = components.NewTaskRegistry()
	s.taskRegistry.RegisterHandler(&components.PingHandler{})
	s.taskRegistry.RegisterHandler(&components.WebspeedHandler{})
}

// SendMessage sends a message with given event and data
func (s *SocketIO) SendMessage(event string, data map[string]interface{}) error {
	return s.sendMessage(event, data)
}

// handleEvent processes incoming socket events
func (s *SocketIO) handleEvent(event string, msg interface{}) error {
	var err error

	switch event {
	case "init":
		data := msg.(map[string]interface{})
		pInterval, ok := data["pingInterval"].(float64)
		if !ok {
			log.Error("PingInterval error")
		} else {
			s.keepPing(pInterval)
		}

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

	case "agent-response":
		log.Debug("agent-response")

	default:
		if err = s.handleTaskRequest(event, msg); err != nil {
			log.Infof("Default event: [%v] %+v\n", event, msg)
		}
	}
	return err
}

// handleTaskRequest processes task request events with "request-" prefix
func (s *SocketIO) handleTaskRequest(event string, msg interface{}) error {
	if len(event) < 8 || event[:8] != "request-" {
		return fmt.Errorf("not a task request event")
	}

	// Extract task type by removing "request-" prefix
	taskType := event[8:]
	log.Debugf("task taskType %+v", taskType)
	handler, exists := s.taskRegistry.GetHandler(taskType)
	if !exists {
		return fmt.Errorf("unknown task type: %s", taskType)
	}
	log.Debugf("task handler %+v", handler)

	data, ok := msg.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid data format for task %s", taskType)
	}

	// Validate data format
	if err := handler.ValidateData(data); err != nil {
		return fmt.Errorf("data validation failed for task %s: %v", taskType, err)
	}

	// Preprocess data
	processedData, response, err := handler.PreProcess(data)
	if err != nil {
		return fmt.Errorf("data preprocessing failed for task %s: %v", taskType, err)
	}

	// Send preprocessing response
	s.sendMessage("agent-response", response)

	// Execute task
	taskId := data["taskId"].(string)
	go s.executeTask(handler, processedData, taskId)

	return nil
}

// executeTask executes a task with the given handler and data
func (s *SocketIO) executeTask(handler components.TaskHandler, data map[string]interface{}, taskId string) {
	// Create stop channel
	s.taskChanDone[taskId] = make(chan struct{})

	defer func() {
		if s.taskChanDone[taskId] != nil {
			select {
			case <-s.taskChanDone[taskId]:
			default:
				close(s.taskChanDone[taskId])
			}
			delete(s.taskChanDone, taskId)
		}
		if r := recover(); r != nil {
			log.Debugf("Task %s panic: %v", taskId, r)
		}
	}()

	// Execute task
	err := handler.Execute(data, taskId, s.taskChanDone[taskId], s)
	if err != nil {
		log.Debugf("Task %s execution failed: %v", taskId, err)
	}
}

// keepPing maintains ping heartbeat at specified interval
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
				s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(`2/agent`)}
			}
		}
	}(pingInterval)
}

// keepAlive sends periodic keep-alive messages every 10 seconds
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

// delay implements exponential backoff delay for connection retries
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

// sayHello sends initial authentication message with token
func (s *SocketIO) sayHello() {
	eventName := "agent-task"
	eventData := map[string]interface{}{
		"token": s.token,
	}
	message, _ := s.escapedString(eventName, eventData)
	s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(message)}
}