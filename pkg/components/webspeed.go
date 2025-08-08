// Copyright 2024-2025 Admin.IM <dev@admin.im>
// SPDX-License-Identifier: GPL-3.0-or-later

package components

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptrace"
	neturl "net/url"
	"strings"
	"time"

	"github.com/admuu/adm-agent/pkg/network"
	"github.com/admuu/adm-agent/pkg/utils"
)

var log = utils.GetLogger()

// WebspeedHandler handles web speed test tasks
type WebspeedHandler struct{}

// ValidateData validates task data format
func (wh *WebspeedHandler) ValidateData(data map[string]interface{}) error {
	if data["content"] == nil || data["type"] == nil || data["taskId"] == nil {
		return fmt.Errorf("event data format invalid: missing required fields")
	}
	if clientIP := data["clientIP"]; clientIP != nil {
		log.Infof("%v Webspeed\n", clientIP)
	}
	return nil
}

// PreProcess preprocesses task data
func (wh *WebspeedHandler) PreProcess(data map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	content := strings.Trim(data["content"].(string), " \n\"'")
	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		content = strings.Trim(content, "[]")
	}

	ip, _, port, ipVersion, err := network.FilterIP(content)
	if err != nil {
		return nil, nil, fmt.Errorf("filterIP error: %v", err)
	}

	taskId := data["taskId"].(string)

	// Prepare processed data
	processedData := make(map[string]interface{})
	for k, v := range data {
		processedData[k] = v
	}
	processedData["ip"] = ip
	processedData["url"] = content

	// Prepare response data
	response := map[string]interface{}{
		"ip":        ip,
		"port":      port,
		"ipVersion": ipVersion,
		"taskType":  wh.GetTaskType(),
		"taskId":    taskId,
	}

	return processedData, response, nil
}

// Execute performs web speed test task
func (wh *WebspeedHandler) Execute(data map[string]interface{}, taskId string, stopChan <-chan struct{}, responseSender ResponseSender) error {
	select {
	case <-stopChan:
		return fmt.Errorf("task %v received stop signal", taskId)
	default:
		url := data["url"].(string)
		ip := data["ip"].(string)

		result, err := wh.webSpeedTest(url, ip)
		if err != nil {
			// Send error response
			errorRes := map[string]interface{}{
				"error":    err.Error(),
				"taskType": wh.GetTaskType(),
				"taskId":   taskId,
			}
			return responseSender.SendMessage("agent-response", errorRes)
		}

		// Send success response
		res := map[string]interface{}{
			"httpCode":      result.HTTPCode,
			"totalTime":     result.TotalTime,
			"dnsTime":       result.DNSTime,
			"connectTime":   result.ConnectTime,
			"sslTime":       result.SSLTime,
			"waitTime":      result.WaitTime,
			"downloadTime":  result.DownloadTime,
			"downloadSize":  result.DownloadSize,
			"downloadSpeed": result.DownloadSpeed,
			"redirectCount": result.RedirectCount,
			"redirectTime":  result.RedirectTime,
			"httpHeaders":   result.HTTPHeaders,
			"taskType":      wh.GetTaskType(),
			"taskId":        taskId,
		}
		return responseSender.SendMessage("agent-response", res)
	}
}

// GetTaskType returns task type identifier
func (wh *WebspeedHandler) GetTaskType() string {
	return "webspeed"
}

// WebSpeedTestResult contains web speed test results
type WebSpeedTestResult struct {
	HTTPCode      int     `json:"httpCode"`
	TotalTime     float64 `json:"totalTime"`
	DNSTime       float64 `json:"dnsTime"`
	ConnectTime   float64 `json:"connectTime"`
	SSLTime       float64 `json:"sslTime"`
	WaitTime      float64 `json:"waitTime"`
	DownloadTime  float64 `json:"downloadTime"`
	DownloadSize  int64   `json:"downloadSize"`
	DownloadSpeed float64 `json:"downloadSpeed"`
	RedirectCount int     `json:"redirectCount"`
	RedirectTime  float64 `json:"redirectTime"`
	HTTPHeaders   string  `json:"httpHeaders"`
}

// roundToDecimal rounds value to specified decimal places
func roundToDecimal(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier
}

// getRandomUserAgent returns a random user agent string
func getRandomUserAgent() string {
	userAgents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
        "Mozilla/5.0 (Linux; Android 14; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
	}
	return userAgents[rand.Intn(len(userAgents))]
}

// RedirectCapturingRoundTripper captures redirect responses and headers
type RedirectCapturingRoundTripper struct {
	Transport     http.RoundTripper
	AllHeaders    *strings.Builder
	RedirectCount int
}

func (rt *RedirectCapturingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.Transport.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// Capture redirect response headers
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		rt.RedirectCount++
		rt.AllHeaders.WriteString(fmt.Sprintf("HTTP/%d.%d %d %s\n",
			resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, http.StatusText(resp.StatusCode)))

		for key, values := range resp.Header {
			for _, value := range values {
				rt.AllHeaders.WriteString(fmt.Sprintf("%s: %s\n", key, value))
			}
		}
		rt.AllHeaders.WriteString("\n")
	}
	return resp, err
}

// webSpeedTest performs the actual web speed test
func (wh *WebspeedHandler) webSpeedTest(url, targetIP string) (*WebSpeedTestResult, error) {
	const (
		maxRedirects    = 5
		connectTimeout  = 2 * time.Second
		totalTimeout    = 10 * time.Second
		maxDownloadSize = 2 * 1024 * 1024
		maxDownloadTime = 8 * time.Second
	)

	result := &WebSpeedTestResult{HTTPHeaders: ""}

	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return nil, fmt.Errorf("parse URL error: %v", err)
	}

	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		if parsedURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// DNS lookup timing
	dnsStart := time.Now()
	_, dnsErr := net.LookupIP(host)
	dnsEnd := time.Now()
	if dnsErr != nil {
		log.Debugf("DNS lookup warning: %v", dnsErr)
	}

	dialer := &net.Dialer{Timeout: connectTimeout}
	var allHttpHeaders strings.Builder

	// Setup custom transport with IP override
	baseTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if strings.Contains(addr, host) {
				ip := targetIP
				if strings.Contains(ip, ":") && !strings.HasPrefix(ip, "[") {
					ip = "[" + ip + "]"
				}
				addr = strings.Replace(addr, host, ip, 1)
			}
			return dialer.DialContext(ctx, network, addr)
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: connectTimeout,
	}

	customTransport := &RedirectCapturingRoundTripper{
		Transport:     baseTransport,
		AllHeaders:    &allHttpHeaders,
		RedirectCount: 0,
	}

	var redirectStartTime time.Time
	var redirectTotalTime float64

	// Setup HTTP client with redirect handling
	client := &http.Client{
		Transport: customTransport,
		Timeout:   totalTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("too many redirects")
			}
			if len(via) == 1 {
				redirectStartTime = time.Now()
			}
			return nil
		},
	}

	// Setup timing variables for tracing
	var connectStart, sslStart time.Time
	var connectEnd, sslEnd time.Time
	var firstByteTime time.Time

	// HTTP trace for detailed timing
	trace := &httptrace.ClientTrace{
		ConnectStart: func(network, addr string) { connectStart = time.Now() },
		ConnectDone:  func(network, addr string, err error) { connectEnd = time.Now() },
		TLSHandshakeStart: func() { sslStart = time.Now() },
		TLSHandshakeDone:  func(state tls.ConnectionState, err error) { sslEnd = time.Now() },
		GotFirstResponseByte: func() { firstByteTime = time.Now() },
	}

	// Create and configure request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request error: %v", err)
	}
	req = req.WithContext(httptrace.WithClientTrace(context.Background(), trace))

	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("User-Agent", getRandomUserAgent())
	req.Host = host

	// Execute request and measure total time
	totalStart := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	result.RedirectCount = customTransport.RedirectCount
	result.HTTPCode = resp.StatusCode

	// Calculate redirect time
	if result.RedirectCount > 0 && !redirectStartTime.IsZero() {
		redirectEndTime := time.Now()
		if !firstByteTime.IsZero() {
			redirectEndTime = firstByteTime
		}
		redirectTotalTime = float64(redirectEndTime.Sub(redirectStartTime).Nanoseconds()) / 1e6
	}

	// Determine download start time
	downloadStart := time.Now()
	if !firstByteTime.IsZero() {
		downloadStart = firstByteTime
	}

	// Capture final response headers
	allHttpHeaders.WriteString(fmt.Sprintf("HTTP/%d.%d %d %s\n",
		resp.ProtoMajor, resp.ProtoMinor, resp.StatusCode, http.StatusText(resp.StatusCode)))

	for key, values := range resp.Header {
		for _, value := range values {
			allHttpHeaders.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
	}

	result.HTTPHeaders = allHttpHeaders.String()

	// Download data with size and time limits
	buffer := make([]byte, 8192)
	var totalDownloaded int64

	downloadCtx, downloadCancel := context.WithTimeout(context.Background(), maxDownloadTime)
	defer downloadCancel()

	downloadDone := make(chan struct{})
	var downloadErr error

	go func() {
		defer close(downloadDone)
		for {
			select {
			case <-downloadCtx.Done():
				return
			default:
			}

			n, err := resp.Body.Read(buffer)
			if n > 0 {
				if totalDownloaded+int64(n) > maxDownloadSize {
					remainingBytes := maxDownloadSize - totalDownloaded
					totalDownloaded += remainingBytes
					return
				}
				totalDownloaded += int64(n)
			}

			if err != nil {
				if err == io.EOF {
					return
				}
				downloadErr = err
				return
			}
		}
	}()

	<-downloadDone
	downloadEnd := time.Now()
	totalEnd := downloadEnd

	if downloadErr != nil && downloadErr != io.EOF {
		log.Debugf("Download warning: %v", downloadErr)
	}

	// Calculate timing metrics
	totalTime := float64(totalEnd.Sub(totalStart).Nanoseconds()) / 1e6
	downloadTime := float64(downloadEnd.Sub(downloadStart).Nanoseconds()) / 1e6
	dnsTime := float64(dnsEnd.Sub(dnsStart).Nanoseconds()) / 1e6

	connectTime := 0.0
	if !connectEnd.IsZero() && !connectStart.IsZero() {
		connectTime = float64(connectEnd.Sub(connectStart).Nanoseconds()) / 1e6
	}

	sslTime := 0.0
	if !sslEnd.IsZero() && !sslStart.IsZero() {
		sslTime = float64(sslEnd.Sub(sslStart).Nanoseconds()) / 1e6
	}

	waitTime := 0.0
	if !firstByteTime.IsZero() {
		var requestStartTime time.Time
		if !sslEnd.IsZero() {
			requestStartTime = sslEnd
		} else if !connectEnd.IsZero() {
			requestStartTime = connectEnd
		} else {
			requestStartTime = totalStart
		}
		waitTime = float64(firstByteTime.Sub(requestStartTime).Nanoseconds()) / 1e6
	}

	// Calculate download speed (bytes per second)
	downloadSpeed := 0.0
	if downloadTime > 0 {
		downloadSpeed = float64(totalDownloaded) / ((waitTime + downloadTime) / 1000.0)
	}

	// Set results with rounded values
	result.TotalTime = roundToDecimal(totalTime, 3)
	result.DownloadTime = roundToDecimal(downloadTime, 3)
	result.DNSTime = roundToDecimal(dnsTime, 3)
	result.ConnectTime = roundToDecimal(connectTime, 3)
	result.SSLTime = roundToDecimal(sslTime, 3)
	result.WaitTime = roundToDecimal(waitTime, 3)
	result.RedirectTime = roundToDecimal(redirectTotalTime, 3)
	result.DownloadSize = totalDownloaded
	result.DownloadSpeed = roundToDecimal(downloadSpeed, 2)

	return result, nil
}