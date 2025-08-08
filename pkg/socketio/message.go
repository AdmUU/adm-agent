// Copyright 2024-2025 Admin.IM <dev@admin.im>
// SPDX-License-Identifier: GPL-3.0-or-later

package socketio

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/admuu/adm-agent/pkg/utils"
	"github.com/gorilla/websocket"
)

// SocketMessage represents a parsed socket.io message
type SocketMessage struct {
	Type  int
	Event string
	Data  interface{}
}

// EventMessage represents an event-based message structure
type EventMessage struct {
	EventType string      `json:"0"`
	Data      interface{} `json:"1"`
}

// websocketWriter handles writing messages to the websocket connection
func (s *SocketIO) websocketWriter() {
	for message := range s.messageChan {
		err := s.conn.WriteMessage(message.messageType, message.data)
		if err != nil {
			log.Errorf("Error writing message: %v", err)
			s.conn.Close()
			break
		}
	}
}

// sendMessage sends an event message through the websocket
func (s *SocketIO) sendMessage(event string, data map[string]interface{}) error {
	eventData := map[string]interface{}{
		"res": data,
	}
	message, err := s.escapedString(event, eventData)
	if err != nil {
		log.Info("escapedString Error:", err)
		return nil
	}
	log.Debugf("Event: %s, Message: %s", event, message)
	select {
	case s.messageChan <- WebSocketMessage{websocket.TextMessage, []byte(message)}:
	case <-time.After(3 * time.Second):
		log.Warn("Dropping message")
		return nil
	}
	return nil
}

// getSchemeHost extracts websocket scheme and host from API URL
func (s *SocketIO) getSchemeHost() (string, string, error) {
	parsedURL, err := url.Parse(s.ApiUrl)
	if err != nil {
		return "", "", fmt.Errorf("failed to parse URL: %v", err)
	}
	scheme := ""
	if parsedURL.Scheme == "https" {
		scheme = "wss"
	} else if parsedURL.Scheme == "http" {
		scheme = "ws"
	} else {
		return "", "", fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}
	return scheme, parsedURL.Host, nil
}

// escapedString formats event data into socket.io message format
func (s *SocketIO) escapedString(eventName string, eventData interface{}) (string, error) {
	escapedJSON, err := utils.ToJSON(eventData)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("42/agent,[\"%s\",%s]", eventName, escapedJSON), nil
}

// parseSocketMessage parses incoming socket.io messages based on format patterns
func (s *SocketIO) parseSocketMessage(bytemsg []byte) (SocketMessage, error) {
	var err error
	msg := string(bytemsg)

	// Define regex patterns for different message types
	eventjsonMessageRegex := regexp.MustCompile(`^\d+(/agent,)?\["[^"]+",\{.*\}\]$`)
	eventStringMessageRegex := regexp.MustCompile(`^\d+(/agent,)?\["[^"]+",".*"\]$`)
	initMessageRegex := regexp.MustCompile(`^\d+\{.*\}$`)
	heartbeatRegex := regexp.MustCompile(`^\d+[^\d\[]*$`)

	switch {
	case eventjsonMessageRegex.MatchString(msg):
		// Parse JSON event messages
		jsonStartIndex := regexp.MustCompile(`\["`).FindStringIndex(msg)[0]
		eventName, eventData, err := s.parseEventMessage(msg[jsonStartIndex:])
		if err != nil {
			return SocketMessage{}, err
		}
		return SocketMessage{
			Type:  1,
			Event: eventName,
			Data:  eventData,
		}, nil

	case eventStringMessageRegex.MatchString(msg):
		// Parse string event messages
		jsonStartIndex := regexp.MustCompile(`\["`).FindStringIndex(msg)[0]
		event, err := s.parseEventMessage2(msg[jsonStartIndex:])
		if err != nil {
			return SocketMessage{}, err
		}
		return SocketMessage{
			Type:  1,
			Event: event.EventType,
			Data:  event.Data,
		}, nil

	case initMessageRegex.MatchString(msg):
		// Parse initialization messages
		jsonStartIndex := regexp.MustCompile(`\{`).FindStringIndex(msg)[0]
		initMessage, err := s.parseInitMessage(msg[jsonStartIndex:])
		if err != nil {
			return SocketMessage{}, err
		}
		return SocketMessage{
			Type:  1,
			Event: "init",
			Data:  initMessage,
		}, nil

	case heartbeatRegex.MatchString(msg):
		// Handle heartbeat messages
		s.heartbeatTime = time.Now()
		heartbeatNumRegex := regexp.MustCompile(`^\d+`)
		msgCodeMatch := heartbeatNumRegex.FindString(msg)
		msgCode, _ := strconv.Atoi(msgCodeMatch)
		if msgCode == 41 {
			return SocketMessage{
				Type:  1,
				Event: "close",
				Data:  nil,
			}, nil
		}
	default:
		err = fmt.Errorf("invalid message format: %s", msg)
	}
	return SocketMessage{}, err
}

// parseEventMessage parses JSON-format event messages
func (s *SocketIO) parseEventMessage(msg string) (string, interface{}, error) {
	msg = msg[1 : len(msg)-1] // Remove brackets
	parts := regexp.MustCompile(`,`).Split(msg, 2)
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("invalid event message format")
	}
	eventName := parts[0][1 : len(parts[0])-1] // Remove quotes

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(parts[1]), &data); err != nil {
		return "", nil, fmt.Errorf("error parsing JSON data:%s", err)
	}
	return eventName, data, nil
}

// parseEventMessage2 parses array-format event messages
func (s *SocketIO) parseEventMessage2(msg string) (EventMessage, error) {
	var event EventMessage
	var rawData []interface{}
	err := json.Unmarshal([]byte(msg), &rawData)
	if err != nil {
		return event, fmt.Errorf("error parsing EventMessage: %s", err)
	}
	if len(rawData) != 2 {
		return event, fmt.Errorf("unsupported EventMessage format : %s", err)
	}
	event = EventMessage{
		EventType: rawData[0].(string),
		Data:      rawData[1],
	}
	return event, nil
}

// parseInitMessage parses initialization JSON messages
func (s *SocketIO) parseInitMessage(msg string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(msg), &data); err != nil {
		return data, fmt.Errorf("parseInitMessage error:%s", err)
	}
	return data, nil
}