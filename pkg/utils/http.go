/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/

package utils

import (
	"bytes"
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type Http struct {
	Url         string
	Method      string
	Data        interface{}
	Timeout     time.Duration
	Jar         *cookiejar.Jar
	NetworkType interface{}
	Certificate Certificate
}

type Response struct {
	RequestID   *string        `json:"requestId,omitempty"`
	Path        *string        `json:"path,omitempty"`
	Success     bool           `json:"success"`
	Message     string         `json:"message"`
	Code        int            `json:"code"`
	Data        interface{}    `json:"data,omitempty"`
	Jar         *cookiejar.Jar `json:"-"`
}

type Certificate struct {
    CertPem []byte 
    CertKey []byte 
}

var log = GetLogger()

func (h *Http) client() *http.Client {
	if h.Timeout == 0 {
		h.Timeout = 30
	}

	dialer := &net.Dialer{
		Timeout:   h.Timeout * time.Second,
		KeepAlive: h.Timeout * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, addr string) (net.Conn, error) {
			if h.NetworkType == nil {
				h.NetworkType = "tcp"
			}
			return dialer.DialContext(ctx, h.NetworkType.(string), addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	
	if len(h.Certificate.CertPem) > 0 && len(h.Certificate.CertKey) > 0 {
		cert, err := tls.X509KeyPair(h.Certificate.CertPem, h.Certificate.CertKey)
		if err != nil {
			log.Fatalf("Error loading certificate and key: %v", err)
		}
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}
		transport.TLSClientConfig = tlsConfig
	}
	
	return &http.Client{
		Transport: transport,
	}
}

func (h *Http) do() (*http.Response, *cookiejar.Jar, error) {
	var req *http.Request
	var resp *http.Response
	var jar *cookiejar.Jar
	var err error

	switch h.Method {
	case "GET":
		if h.Data != nil {
			h.Url += "?" + h.encodeParams()
		}
		req, err = http.NewRequest(h.Method, h.Url, nil)
	case "POST", "PUT", "PATCH":
		var body io.Reader
		if h.Data != nil {
			body, err = h.encodeBody()
			if err != nil {
				return nil, nil, err
			}
		}
		req, err = http.NewRequest(h.Method, h.Url, body)
		if err == nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}
	default:
		req, err = http.NewRequest(h.Method, h.Url, nil)
	}
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("accept-language", "en")
	jar, _ = cookiejar.New(nil)

	client := h.client()
	client.Jar = jar
	resp, err = client.Do(req)
	return resp, jar, err
}

func (h *Http) ApiRequest() (Response, error) {
	var response Response
	var err error

	resp, jar, err := h.do()
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	log.Debug("Response Body:", string(body))

	err = json.Unmarshal([]byte(string(body)), &response)
	if err != nil {
		return response, err
	}

	if response.Code != 200 {
		return response, fmt.Errorf("[%v] %v", response.Code, response.Message)
	}
	response.Jar = jar
	return response, nil
}

func (h *Http) UrlRequest() (string, error) {
	var response string
	var err error

	resp, _, err := h.do()
	if err != nil {
		return response, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	log.Debug("Response Body:", string(body))

	response = string(body)
	return response, nil
}

func (h *Http) encodeParams() string {
	values := url.Values{}
	if params, ok := h.Data.(map[string]string); ok {
		for key, value := range params {
			values.Add(key, value)
		}
	}
	return values.Encode()
}

func (h *Http) encodeBody() (io.Reader, error) {
	switch v := h.Data.(type) {
	case string:
		return strings.NewReader(v), nil
	case []byte:
		return bytes.NewReader(v), nil
	case url.Values:
		return strings.NewReader(v.Encode()), nil
	default:
		jsonData, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		return bytes.NewReader(jsonData), nil
	}
}
