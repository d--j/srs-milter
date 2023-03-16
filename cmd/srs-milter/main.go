package main

import (
	"context"
	"flag"
	"log"
	"sync"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/srs-milter"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var RuntimeConfig *srsMilter.Configuration
var RuntimeCache *srsMilter.Cache
var RuntimeConfigMutex sync.RWMutex

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

/* main program */
func main() {
	// parse commandline arguments
	var systemd bool
	var protocol, address, forward, reverse string
	flag.StringVar(&protocol,
		"proto",
		"tcp",
		"Protocol family (unix or tcp)")
	flag.StringVar(&address,
		"addr",
		"127.0.0.1:10382",
		"Bind to address/port or unix domain socket path")
	flag.StringVar(&forward,
		"forward",
		"",
		"`email` to do forward SRS lookup for. If specified the milter will not be started.")
	flag.StringVar(&reverse,
		"reverse",
		"",
		"`email` to do reverse SRS lookup for. If specified the milter will not be started.")
	flag.BoolVar(&systemd, "systemd", false, "enable systemd mode (log without date/time)")
	flag.Parse()

	// disable logging date/time when called as systemd service – journald will add those anyway
	if systemd {
		log.Default().SetFlags(0)
	}

	log.Printf("info=\"start\" version=%q commit=%q buildDate=%q", version, commit, date)

	// make sure the specified protocol is either unix or tcp
	if protocol != "unix" && protocol != "tcp" {
		log.Fatal("invalid protocol name")
	}

	var err error
	viper.SetConfigName("srs-milter")
	viper.AddConfigPath("/etc/srs-milter")
	viper.AddConfigPath(".")
	if err = viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	RuntimeConfig, err = loadViperConfig()
	if err != nil {
		log.Fatal(err)
	}
	RuntimeConfig.Setup()
	RuntimeCache = srsMilter.NewCache(RuntimeConfig)

	if forward != "" {
		srsAddress, err := srsMilter.ForwardSrs(forward, RuntimeConfig)
		log.Printf("address=<%s> srsAddress=<%s> error=%v", forward, srsAddress, err)
	}
	if reverse != "" {
		address, err := srsMilter.ReverseSrs(reverse, RuntimeConfig)
		log.Printf("srsAddress=<%s> address=<%s> error=%v", reverse, address, err)
	}
	if forward != "" || reverse != "" {
		return
	}

	viper.OnConfigChange(func(_ fsnotify.Event) {
		newConfig, err := loadViperConfig()
		if err != nil {
			log.Printf("warn=\"could not load new config on change\" error=%q", err)
		} else {
			newConfig.Setup()
			RuntimeConfigMutex.Lock()
			RuntimeConfig = newConfig
			RuntimeCache = srsMilter.NewCache(RuntimeConfig)
			RuntimeConfigMutex.Unlock()
		}
	})
	viper.WatchConfig()

	filter, err := mailfilter.New(protocol, address, func(ctx context.Context, trx mailfilter.Trx) (mailfilter.Decision, error) {
		RuntimeConfigMutex.RLock()
		config := RuntimeConfig
		cache := RuntimeCache
		RuntimeConfigMutex.RUnlock()
		return srsMilter.Filter(ctx, trx, config, cache)
	}, mailfilter.WithDecisionAt(mailfilter.DecisionAtEndOfHeaders))
	if err != nil {
		log.Fatal(err)
	}
	if RuntimeConfig.LogLevel > 0 {
		log.Printf("info=\"ready\" network=%q address=%q", filter.Addr().Network(), filter.Addr().String())
	}

	// quit when milter quits
	filter.Wait()
}
