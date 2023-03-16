package main

import (
	"errors"
	"log"
	"net"
	"reflect"
	"strings"

	"github.com/d--j/srs-milter"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"golang.org/x/net/idna"
)

func determineExternalIPs() ([]net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var ips []net.IP
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() || ip.IsInterfaceLocalMulticast() || ip.IsLinkLocalMulticast() || ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
				continue
			}
			ips = append(ips, ip)
		}
	}
	if len(ips) == 0 {
		return nil, errors.New("could not find public IP addresses. define them in the config file via localIps")
	}
	return ips, nil
}

func ipsToString(ips []net.IP) string {
	var s strings.Builder
	s.WriteString(`"`)
	for i, ip := range ips {
		if i > 0 {
			s.WriteString(",")
		}
		s.WriteString(ip.String())
	}
	s.WriteString(`"`)
	return s.String()
}

func loadViperConfig() (*srsMilter.Configuration, error) {
	var conf srsMilter.Configuration
	err := viper.Unmarshal(&conf, viper.DecodeHook(mapstructure.StringToIPHookFunc()), viper.DecodeHook(func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(srsMilter.Domain("")) {
			return data, nil
		}

		asciiDomain, err := idna.Lookup.ToASCII(data.(string))
		return srsMilter.Domain(asciiDomain), err
	}), viper.DecodeHook(func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t != reflect.TypeOf(net.IP{}) {
			return data, nil
		}

		return net.ParseIP(data.(string)), nil
	}))
	if err != nil {
		return nil, err
	}
	if conf.SrsDomain == "" {
		return nil, errors.New("no srsDomain specified in config file")
	}
	if len(conf.SrsKeys) == 0 {
		return nil, errors.New("no srsKeys specified in config file")
	}
	if len(conf.LocalIps) == 0 {
		conf.LocalIps, err = determineExternalIPs()
		if err != nil {
			return nil, err
		}
	}
	if len(conf.LocalDomains) == 0 {
		log.Printf("warn=\"local domain list is empty: only relying on SPF lookups\"")
	}
	if conf.LogLevel > 0 {
		log.Printf("info=\"config loaded\" srsDomain=%q numSrsKeys=%d numLocalDomains=%d localIPs=%s", conf.SrsDomain, len(conf.SrsKeys), len(conf.LocalDomains), ipsToString(conf.LocalIps))
	}
	return &conf, nil
}
