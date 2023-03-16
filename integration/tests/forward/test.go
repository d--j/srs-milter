package main

import (
	"context"
	"net"

	"github.com/d--j/go-milter/integration"
	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/srs-milter"
	"github.com/d--j/srs-milter/integration/patches"
)

func main() {
	integration.Test(func(ctx context.Context, trx mailfilter.Trx) (mailfilter.Decision, error) {
		p := patches.Apply()
		defer p.Reset()
		config := &srsMilter.Configuration{
			SrsDomain:    srsMilter.ToDomain("srs.example.com"),
			LocalDomains: []srsMilter.Domain{"example.com"},
			SrsKeys:      []string{"secret-key"},
			LocalIps:     []net.IP{net.ParseIP("10.0.0.1")},
			LogLevel:     5,
		}
		config.Setup()
		cache := srsMilter.NewCache(config)
		return srsMilter.Filter(ctx, trx, config, cache)
	}, mailfilter.WithDecisionAt(mailfilter.DecisionAtEndOfHeaders))
}
