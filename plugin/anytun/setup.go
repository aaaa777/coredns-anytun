package socks

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register("anytun", setup)
}

func setup(c *caddy.Controller) error {
	socks, err := parseSocksConfig(c)
	if err != nil {
		return plugin.Error("anytun", err)
	}

	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		socks.Next = next
		return socks
	})

	return nil
}

func parseSocksConfig(c *caddy.Controller) (*SOCKSPlugin, error) {
	socks := &SOCKSPlugin{}

	for c.Next() {
		args := c.RemainingArgs()
		
		if len(args) < 2 {
			return nil, c.Err("anytun requires at least 2 arguments: <dns-server> <domain-suffix>")
		}

		socks.ProxyAddr = args[0]
		socks.DNSServers = args[1:]
	}

	return socks, nil
}