package srsMilter

import (
	"net"

	"golang.org/x/net/idna"
)

type Domain string

func ToDomain(domain string) Domain {
	ascii, err := idna.Lookup.ToASCII(domain)
	if err != nil {
		return Domain(domain)
	}
	return Domain(ascii)
}

func (d Domain) String() string {
	return string(d)
}

func (d Domain) Unicode() string {
	uni, err := idna.Lookup.ToUnicode(string(d))
	if err != nil {
		return string(d)
	}
	return uni
}

type Configuration struct {
	SrsDomain      Domain
	LocalDomains   []Domain
	SrsKeys        []string
	LocalIps       []net.IP
	LogLevel       uint
	localDomainMap map[string]bool
}

func (c *Configuration) Setup() {
	c.localDomainMap = make(map[string]bool)
	for _, d := range c.LocalDomains {
		c.localDomainMap[d.String()] = true
	}
}

func (c *Configuration) HasLocalDomain(asciiDomain string) bool {
	return c.localDomainMap[asciiDomain]
}
