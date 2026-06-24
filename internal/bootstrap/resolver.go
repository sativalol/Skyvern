package bootstrap

import (
	"context"
	"net"
	"strings"
	"time"
)

func init() {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: 2 * time.Second,
			}
			isLocal := strings.Contains(address, "127.0.0.1") || strings.Contains(address, "::1") || strings.Contains(address, "localhost")
			if isLocal {
				for _, pub := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
					c, err := d.DialContext(ctx, network, pub)
					if err == nil {
						return c, nil
					}
				}
			}
			c, err := d.DialContext(ctx, network, address)
			if err == nil {
				return c, nil
			}
			for _, pub := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
				if address == pub {
					continue
				}
				c, err := d.DialContext(ctx, network, pub)
				if err == nil {
					return c, nil
				}
			}
			return nil, err
		},
	}
}
