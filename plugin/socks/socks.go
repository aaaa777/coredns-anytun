package socks

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/miekg/dns"
	"golang.org/x/net/proxy"
)

type SOCKSPlugin struct {
	Next       plugin.Handler
	ProxyAddr  string
	DNSServers []string
}

func (s *SOCKSPlugin) Name() string { return "socks" }

func (s *SOCKSPlugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	// SOCKSプロキシダイアラーを作成
	dialer, err := proxy.SOCKS5("tcp", s.ProxyAddr, nil, proxy.Direct)
	if err != nil {
		return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
	}

	// 設定されたDNSサーバーに順番にクエリを試行
	for _, dnsServer := range s.DNSServers {
		// SOCKSプロキシを使用してDNSサーバーに接続
		conn, err := dialer.Dial("udp", net.JoinHostPort(dnsServer, "53"))
		if err != nil {
			continue
		}
		defer conn.Close()

		// UDPクライアントを作成
		c := &dns.Client{
			Net:     "udp",
			Timeout: 5 * time.Second,
		}

		// DNSクエリを送信
		resp, _, err := c.ExchangeWithConn(r, &dns.Conn{Conn: conn})
		if err != nil {
			continue
		}

		w.WriteMsg(resp)
		return dns.RcodeSuccess, nil
	}

	// すべてのDNSサーバーへの接続に失敗した場合
	return plugin.NextOrFailure(s.Name(), s.Next, ctx, w, r)
}

// プラグインの設定を解析する関数
func (s *SOCKSPlugin) Parse(tokens []string) error {
	if len(tokens) < 3 {
		return fmt.Errorf("socks requires at least 3 arguments: <proxy-addr> <dns-server1> [dns-server2...]")
	}

	s.ProxyAddr = tokens[0]
	s.DNSServers = tokens[1:]

	return nil
}

// プラグインのセットアップ関数
func Setup(c *dnsserver.Config) error {
	return nil
}