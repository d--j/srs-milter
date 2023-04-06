package main

import (
	"context"
	"flag"
	"net"
	"os"
	"sync"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/go-socketmap"
	"github.com/d--j/srs-milter"
	"github.com/fsnotify/fsnotify"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
)

var RuntimeConfig *srsmilter.Configuration
var RuntimeCache *srsmilter.Cache
var RuntimeConfigMutex sync.RWMutex
var LogHandler log15.Handler

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

/* main program */
func main() {
	// parse commandline arguments
	var systemd bool
	var milterProtocol, milterAddress, socketmapProtocol, socketmapAddress, forward, reverse string
	flag.StringVar(&milterProtocol,
		"milterProto",
		"tcp",
		"Protocol `family` (unix or tcp) of milter server")
	flag.StringVar(&milterAddress,
		"milterAddr",
		"127.0.0.1:10382",
		"Bind milter server to `address/port` or unix domain socket path")
	flag.StringVar(&socketmapProtocol,
		"socketmapProto",
		"tcp",
		"Protocol `family` (unix or tcp) of socketmap server")
	flag.StringVar(&socketmapAddress,
		"socketmapAddr",
		"127.0.0.1:10383",
		"Bind socketmap server to `address/port` or unix domain socket path")
	flag.StringVar(&forward,
		"forward",
		"",
		"`email` to do forward SRS lookup for. If specified the daemon will not be started.")
	flag.StringVar(&reverse,
		"reverse",
		"",
		"`email` to do reverse SRS lookup for. If specified the daemon will not be started.")
	flag.BoolVar(&systemd, "systemd", false, "enable systemd mode (log without date/time)")
	flag.Parse()

	// disable logging date/time when called as systemd service – journald will add those anyway
	if systemd {
		LogHandler = log15.StreamHandler(os.Stdout, LogfmtFormatWithoutTime())
	} else {
		LogHandler = log15.StreamHandler(os.Stdout, LogfmtFormatWithTime())
	}
	logger := log15.New()
	logger.SetHandler(LogHandler)

	logger.Info("start", log15.Ctx{"version": version, "commit": commit, "build": date})

	// make sure the specified protocol is either unix or tcp
	if milterProtocol != "unix" && milterProtocol != "tcp" {
		logger.Crit("invalid miler protocol name", "protocol", milterProtocol)
		os.Exit(1)
	}
	if socketmapProtocol != "unix" && socketmapProtocol != "tcp" {
		logger.Crit("invalid socketmap protocol name", "protocol", socketmapProtocol)
		os.Exit(1)
	}

	var err error
	viper.SetConfigName("srs-milter")
	viper.AddConfigPath("/etc/srs-milter")
	viper.AddConfigPath(".")
	if err = viper.ReadInConfig(); err != nil {
		logger.Crit("error reading config file", log15.Ctx{"err": err})
		os.Exit(1)
	}
	RuntimeConfig, err = loadViperConfig()
	if err != nil {
		logger.Crit("error parsing config file", log15.Ctx{"err": err})
		os.Exit(1)
	}
	err = RuntimeConfig.Setup()
	if err != nil {
		logger.Crit("error in config file", log15.Ctx{"err": err})
		os.Exit(1)
	}
	RuntimeCache = srsmilter.NewCache(RuntimeConfig)
	configureLogging := func() {
		if systemd {
			LogHandler = log15.StreamHandler(os.Stdout, LogfmtFormatWithoutTime())
		} else {
			LogHandler = log15.StreamHandler(os.Stdout, LogfmtFormatWithTime())
		}
		switch RuntimeConfig.LogLevel {
		case 0:
			LogHandler = log15.LvlFilterHandler(log15.LvlCrit, LogHandler)
		case 1:
			LogHandler = log15.LvlFilterHandler(log15.LvlError, LogHandler)
		case 2:
			LogHandler = log15.LvlFilterHandler(log15.LvlWarn, LogHandler)
		case 3:
			LogHandler = log15.LvlFilterHandler(log15.LvlInfo, LogHandler)
		default:
			LogHandler = log15.LvlFilterHandler(log15.LvlDebug, LogHandler)
		}
		logger.SetHandler(LogHandler)
		srsmilter.Log.SetHandler(LogHandler)
		logger.Info("config loaded", log15.Ctx{"srsDomain": RuntimeConfig.SrsDomain, "localIps": ipsToString(RuntimeConfig.LocalIps), "numKeys": len(RuntimeConfig.SrsKeys), "numLocalDomains": len(RuntimeConfig.LocalDomains)})
		if len(RuntimeConfig.LocalDomains) == 0 {
			logger.Warn("local domain list is empty: only relying on SPF lookups")
		}
	}
	configureLogging()

	if forward != "" {
		srsAddress, err := srsmilter.ForwardSrs(forward, RuntimeConfig)
		logger.Info("forward SRS", log15.Ctx{"ofrom": forward, "from": srsAddress, "err": err})
	}
	if reverse != "" {
		address, err := srsmilter.ReverseSrs(reverse, RuntimeConfig)
		logger.Info("reverse SRS", log15.Ctx{"oto": reverse, "to": address, "err": err})
	}
	if forward != "" || reverse != "" {
		return
	}

	viper.OnConfigChange(func(_ fsnotify.Event) {
		newConfig, err := loadViperConfig()
		if err != nil {
			logger.Error("could not load new config on change", "err", err)
		} else {
			err = newConfig.Setup()
			if err != nil {
				logger.Error("could not load new config on change", "err", err)
			}
			RuntimeConfigMutex.Lock()
			RuntimeConfig = newConfig
			RuntimeCache = srsmilter.NewCache(RuntimeConfig)
			configureLogging()
			RuntimeConfigMutex.Unlock()
		}
	})
	viper.WatchConfig()

	filter, err := mailfilter.New(milterProtocol, milterAddress, func(ctx context.Context, trx mailfilter.Trx) (mailfilter.Decision, error) {
		RuntimeConfigMutex.RLock()
		config := RuntimeConfig
		cache := RuntimeCache
		RuntimeConfigMutex.RUnlock()
		return srsmilter.Filter(ctx, trx, config, cache)
	}, mailfilter.WithDecisionAt(mailfilter.DecisionAtEndOfHeaders))
	if err != nil {
		logger.Crit("error creating milter", "err", err)
		os.Exit(1)
	}

	smListener, err := net.Listen(socketmapProtocol, socketmapAddress)
	if err != nil {
		logger.Crit("error creating socketmap listener", "err", err)
		os.Exit(1)
	}

	go func() {
		socketmap.Serve(smListener, func(_ context.Context, lookup, key string) (string, bool, error) {
			RuntimeConfigMutex.RLock()
			config := RuntimeConfig
			RuntimeConfigMutex.RUnlock()
			return srsmilter.Socketmap(config, lookup, key)
		})
	}()

	logger.Info("ready", "milterProto", filter.Addr().Network(), "milterAddr", filter.Addr().String(), "socketmapProto", smListener.Addr().Network(), "socketmapAddr", smListener.Addr().String())

	// quit when milter quits
	filter.Wait()
}
