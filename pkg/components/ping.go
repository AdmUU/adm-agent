// Copyright 2024-2025 Admin.IM <dev@admin.im>
// SPDX-License-Identifier: GPL-3.0-or-later

package components

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/admuu/adm-agent/pkg/network"
	"github.com/go-ping/ping"
)

// PingHandler handles ping task operations
type PingHandler struct{}

// ValidateData checks if required fields are present in the data
func (ph *PingHandler) ValidateData(data map[string]interface{}) error {
	if data["host"] == nil || data["pingtype"] == nil || data["protocol"] == nil || data["taskId"] == nil {
		return fmt.Errorf("event data format invalid: missing required fields")
	}
	if clientIP := data["clientIP"]; clientIP != nil {
		log.Infof("%v Ping %v %v\n", clientIP, data["protocol"], data["pingtype"])
	}
	return nil
}

// PreProcess prepares data for ping execution and creates response structure
func (ph *PingHandler) PreProcess(data map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	// Clean host string and remove brackets for IPv6
	host := strings.Trim(data["host"].(string), " \n\"'")
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = strings.Trim(host, "[]")
	}

	// Parse IP address and extract components
	ip, host, port, ipVersion, err := network.FilterIP(host)
	if err != nil {
		return nil, nil, fmt.Errorf("filterIP error: %v", err)
	}

	protocol := data["protocol"].(string)
	taskId := data["taskId"].(string)

	// Clear port for non-TCP protocols
	if protocol != "tcp" {
		port = ""
	}

	// Create processed data with additional fields
	processedData := make(map[string]interface{})
	for k, v := range data {
		processedData[k] = v
	}
	processedData["ip"] = ip
	processedData["host"] = host
	processedData["port"] = port
	processedData["ipVersion"] = ipVersion

	// Create response data structure
	response := map[string]interface{}{
		"ip":        ip,
		"port":      port,
		"ipVersion": ipVersion,
		"taskType":  ph.GetTaskType(),
		"taskId":    taskId,
	}

	return processedData, response, nil
}

// Execute performs ping operations based on protocol and ping type
func (ph *PingHandler) Execute(data map[string]interface{}, taskId string, stopChan <-chan struct{}, responseSender ResponseSender) error {
	var pingCount = 3
	var loopCount = 1
	var delay float32
	var err error

	ip := data["ip"].(string)
	protocol := data["protocol"].(string)
	pingtype := data["pingtype"].(string)

	// Set parameters for continuous ping
	if pingtype == "continuous" {
		pingCount = 1
		loopCount = 100
	}

	// Execute ping operations
	for i := 0; i < loopCount; i++ {
		select {
		case <-stopChan:
			return fmt.Errorf("task %v received stop signal", taskId)
		default:
			startTime := time.Now()

			// Choose ping method based on protocol
			if protocol == "icmp" {
				delay, err = IcmpPing(ip, pingCount)
			} else if protocol == "tcp" {
				delay, err = TcpPing(data)
			}

			// Set delay to 0 on error
			if err != nil {
				delay = 0
			}

			// Send response with delay result
			res := map[string]interface{}{
				"delay":    delay,
				"taskType": ph.GetTaskType(),
				"taskId":   taskId,
			}

			if sendErr := responseSender.SendMessage("agent-response", res); sendErr != nil {
				return sendErr
			}

			// Ensure minimum 1 second interval between pings
			duration := time.Since(startTime)
			if duration < 1*time.Second {
				remainingTime := 1*time.Second - duration
				time.Sleep(remainingTime)
			}
		}
	}
	return nil
}

// GetTaskType returns the task type identifier
func (ph *PingHandler) GetTaskType() string {
	return "ping"
}

// IcmpPing performs ICMP ping and returns average delay in milliseconds
func IcmpPing(ip string, count int) (float32, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return 0, err
	}

	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = 800 * time.Millisecond

	err = pinger.Run()
	if err != nil {
		return 0, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, errors.New("PacketsRecv 0")
	}

	// Convert average RTT from microseconds to milliseconds
	delay := float32(stats.AvgRtt.Microseconds()) / 1000.0
	return delay, nil
}

// TcpPing performs TCP connection test and returns delay in milliseconds
func TcpPing(data map[string]interface{}) (float32, error) {
	ip := data["ip"].(string)
	port := data["port"].(string)

	startTime := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), 800*time.Millisecond)
	if err != nil {
		return 0, err
	}
	conn.Close()

	// Convert elapsed time from microseconds to milliseconds
	delay := float32(time.Since(startTime).Microseconds()) / 1000.0
	return delay, nil
}