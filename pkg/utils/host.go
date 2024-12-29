/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

func GenerateFingerprint() string {
	var info strings.Builder

	hostInfo, _ := host.Info()
	info.WriteString(hostInfo.Hostname)
	info.WriteString(hostInfo.Platform)
	info.WriteString(hostInfo.PlatformVersion)
	cpuInfo, _ := cpu.Info()
	if len(cpuInfo) > 0 {
		info.WriteString(cpuInfo[0].ModelName)
	}
	memInfo, _ := mem.VirtualMemory()
	info.WriteString(fmt.Sprintf("%d", memInfo.Total))
	info.WriteString(getMACAddress())
	info.WriteString(runtime.GOOS)
	info.WriteString(runtime.GOARCH)
	hash := sha256.Sum256([]byte(info.String()))
	return hex.EncodeToString(hash[:])
}

func getMACAddress() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return ""
	}

	for _, i := range interfaces {
		if i.Flags&net.FlagUp != 0 && !strings.HasPrefix(i.Name, "lo") {
			addrs, err := i.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				var ip net.IP
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil || ip.IsLoopback() {
					continue
				}
				return i.HardwareAddr.String()
			}
		}
	}
	return ""
}
