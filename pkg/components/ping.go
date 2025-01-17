/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package components

import (
	"errors"
	"net"
	"time"

	"github.com/go-ping/ping"
)

func IcmpPing(ip string, count int) (float32, error) {
	pinger, err := ping.NewPinger(ip)
	if err != nil {
		return 0, err
	}
	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = 800*time.Millisecond
	err = pinger.Run()
	if err != nil {
		return 0, err
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv == 0 {
		return 0, errors.New("PacketsRecv 0")
	}
	delay := float32(stats.AvgRtt.Microseconds()) / 1000.0
	return delay, nil
}

func TcpPing(data map[string]interface{}) (float32, error) {
	ip := data["ip"].(string)
	port := data["port"].(string)
	startTime := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, port), 800*time.Millisecond)
	if err != nil {
		return 0, err
	}
	conn.Close()
	delay := float32(time.Since(startTime).Microseconds()) / 1000.0
	return delay, nil
}
