package util

import (
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// GetLocalIPs returns all active local interface IPv4/IPv6 addresses (excluding loopback)
func GetLocalIPs() []string {
	ips := make([]string, 0)
	interfaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				ipStr := addr.String()
				if strings.Contains(ipStr, ".") {
					ipStr = strings.Split(ipStr, "/")[0]
					ips = append(ips, ipStr)
				} else if strings.HasPrefix(ipStr, "fe80::") {
					continue
				} else {
					ipStr = strings.Split(ipStr, "/")[0]
					ips = append(ips, ipStr)
				}
			}
		}
	}
	return ips
}

// GetPublicIP 通过多个外部API并发检测获取公网IP
func GetPublicIP() string {
	apis := []string{
		"https://ip.me",
		"https://api64.ipify.org",
		"https://ip.sb",
		"https://icanhazip.com",
		"https://ipinfo.io/ip",
		"https://checkip.amazonaws.com",
	}
	type result struct {
		ip  string
		err error
	}
	ch := make(chan result, len(apis))
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 3 * time.Second}

	for _, api := range apis {
		wg.Add(1)
		go func(url string) {
			defer wg.Done()
			resp, err := client.Get(url)
			if err != nil {
				ch <- result{"", err}
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				ch <- result{"", err}
				return
			}
			ch <- result{string(body), nil}
		}(api)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		if res.err == nil && res.ip != "" {
			return strings.TrimSpace(res.ip)
		}
	}
	return ""
}
