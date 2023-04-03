package srsmilter

import (
	"time"

	"blitiri.com.ar/go/spf"
	"github.com/jellydator/ttlcache/v3"
)

func emptyTtlCache() *ttlcache.Cache[string, bool] {
	return ttlcache.New[string, bool](
		ttlcache.WithCapacity[string, bool](1*1024*1024*1024),
		ttlcache.WithTTL[string, bool](30*time.Minute),
		ttlcache.WithDisableTouchOnHit[string, bool](),
	)
}

type Cache struct {
	conf  *Configuration
	cache *ttlcache.Cache[string, bool]
}

func NewCache(conf *Configuration) *Cache {
	return &Cache{
		conf:  conf,
		cache: emptyTtlCache(),
	}
}

func (c *Cache) IsLocalNotAllowedToSend(addr, asciiDomain string) bool {
	if res := c.cache.Get(asciiDomain); res != nil {
		return res.Value()
	}
	// Check if we are not authorized to send for `addr.Addr`
	for _, ip := range c.conf.LocalIps {
		result, _ := spf.CheckHostWithSender(ip, asciiDomain, addr)
		// We rewrite when any of our IPs is not allowed to send
		if result == spf.Fail || result == spf.SoftFail {
			c.Set(asciiDomain, true)
			return true
		}
		// if SPF record is empty or broken we quit early since checking with other IPs will not change result
		if result == spf.None || result == spf.PermError {
			break
		}
	}
	c.Set(asciiDomain, false)
	return false
}

func (c *Cache) Set(asciiDomain string, isLocalNotAllowedToSend bool) {
	c.cache.Set(asciiDomain, isLocalNotAllowedToSend, ttlcache.DefaultTTL)
}
