package dualforwardsocks

import (
    "context"
	"net"
	"time"

	"golang.org/x/net/proxy"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
    "github.com/coredns/coredns/plugin"
    // "github.com/coredns/coredns/request"
    "github.com/miekg/dns"
)

type DualForward struct {
    Primary   string
    Secondary string
    SocksAddr string
    SocksPort string
    Next      plugin.Handler
}

func (df *DualForward) Name() string { return "dualforwardsocks" }

func (df *DualForward) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
    // 最初のDNSサーバーにクエリを送信(するだけ)
    primaryClient := new(dns.Client)
    primaryMsg := r.Copy()
    primaryClient.Exchange(primaryMsg, df.Primary)

	// SOCKSプロキシダイアラーを作成
	dialer, err := proxy.SOCKS5("udp", df.SocksAddr, nil, proxy.Direct)
	if err != nil {
		return plugin.NextOrFailure(df.Name(), df.Next, ctx, w, r)
	}

	// 設定されたDNSサーバーに順番にクエリを試行
    dnsServer := df.Secondary
    // SOCKSプロキシを使用してDNSサーバーに接続
    conn, err := dialer.Dial("udp", net.JoinHostPort(dnsServer, "53"))
    if err != nil {
        return plugin.NextOrFailure(df.Name(), df.Next, ctx, w, r)
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
        // DNSサーバーへの接続に失敗した場合
        return plugin.NextOrFailure(df.Name(), df.Next, ctx, w, r)
    }

    w.WriteMsg(resp)
    return dns.RcodeSuccess, nil
}

// プラグイン設定用の構造体とパース関数
type dualforwardConfig struct {
    Primary   string
    Secondary string
    SocksAddr string
    SocksPort string
}

func setup(c *caddy.Controller) error {
    config := dualforwardConfig{}
    
    for c.Next() {
        if !c.NextArg() {
            return plugin.Error("dualforward", c.ArgErr())
        }
        config.Primary = c.Val()
        
        if !c.NextArg() {
            return plugin.Error("dualforward", c.ArgErr())
        }
        config.Secondary = c.Val()

        if !c.NextArg() {
            return plugin.Error("dualforward", c.ArgErr())
        }
        config.SocksAddr = c.Val()

        if !c.NextArg() {
            return plugin.Error("dualforward", c.ArgErr())
        }
        config.SocksPort = c.Val()

    }
    
    dw := &DualForward{
        Primary:   config.Primary,
        Secondary: config.Secondary,
        SocksAddr: config.SocksAddr,
        SocksPort: config.SocksPort,
    }
    
    dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
        dw.Next = next
        return dw
    })
    
    return nil
}

func init() {
    plugin.Register("dualforwardsocks", setup)
}