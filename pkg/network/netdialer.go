/*
Copyright © 2024-2025 Admin.IM <dev@admin.im>
*/

package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"sort"
	"time"
)
type NetDialer struct {
    Timeout        time.Duration
    KeepAlive      time.Duration
    fallbackDelay  time.Duration
    resolver       *net.Resolver
    preferIPv4     *bool
}

func (d *NetDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {

	if d.preferIPv4 == nil {
		d.preferIPv4 = new(bool)
        *d.preferIPv4 = true
	}

    host, port, err := net.SplitHostPort(addr)
    if err != nil {
        return nil, err
    }

    ips, err := d.resolver.LookupIPAddr(ctx, host)
    if err != nil {
        return nil, err
    }

    if len(ips) == 0 {
        return nil, fmt.Errorf("no IP addresses found for host: %s", host)
    }

    ips = d.sortIPAddrs(ips, *d.preferIPv4)

    var lastErr error
    for _, ip := range ips {
        select {
        case <-ctx.Done():
            return nil, ctx.Err()
        default:
            ipAddr := net.JoinHostPort(ip.String(), port)

            dialer := &net.Dialer{
                Timeout:   d.Timeout,
                KeepAlive: d.KeepAlive,
            }

            conn, err := dialer.DialContext(ctx, network, ipAddr)
            if err == nil {
                if network == "tcp-tls" {
                    tlsConn := tls.Client(conn, &tls.Config{
                        ServerName: host,
                        MinVersion: tls.VersionTLS12,
                    })
                    if err := tlsConn.HandshakeContext(ctx); err != nil {
                        conn.Close()
                        lastErr = err
                        time.Sleep(d.fallbackDelay)
                        continue
                    }
                    return tlsConn, nil
                }
                return conn, nil
            }

            lastErr = err
            time.Sleep(d.fallbackDelay)
        }
    }

    return nil, fmt.Errorf("all IP addresses failed. Last error: %v", lastErr)
}

func (d *NetDialer) sortIPAddrs(ips []net.IPAddr, preferIPv4 bool) []net.IPAddr {
    if !preferIPv4 {
        return ips
    }

    sort.SliceStable(ips, func(i, j int) bool {
        isIPv4i := ips[i].IP.To4() != nil
        isIPv4j := ips[j].IP.To4() != nil

        if isIPv4i != isIPv4j {
            return isIPv4i
        }
        return false
    })

    return ips
}