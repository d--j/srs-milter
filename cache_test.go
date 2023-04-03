package srsmilter

import (
	"net"
	"strings"
	"testing"
	"time"

	"blitiri.com.ar/go/spf"
	"github.com/agiledragon/gomonkey/v2"
)

var ConstantDate = time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)

func monkeyPatch() *gomonkey.Patches {
	// we do not want to go through the hoops of generating a test DNS resolver,
	// just let's patch the library function we use – it has its own unit tests
	return gomonkey.
		ApplyFuncReturn(time.Now, ConstantDate).
		ApplyFunc(spf.CheckHostWithSender, func(_ net.IP, helo, sender string, _ ...spf.Option) (spf.Result, error) {
			if strings.HasSuffix(sender, "@example.com") || helo == "example.com" {
				return spf.Pass, nil
			}
			if strings.HasSuffix(sender, "@example.net") || helo == "example.net" {
				return spf.Fail, nil
			}
			return spf.None, nil
		})
}

func TestCache_IsLocalNotAllowedToSend(t *testing.T) {
	tests := []struct {
		name        string
		addr        string
		asciiDomain string
		want        bool
	}{
		{"local", "someone@example.biz", "example.biz", false},
		{"not local but SPF allowed", "someone@example.com", "example.com", false},
		{"not local and SPF fail", "someone@example.net", "example.net", true},
		{"not local but no SPF", "someone@example.org", "example.org", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCache(&Configuration{LocalDomains: toDomainSlice([]string{"example.biz"}), LocalIps: []net.IP{net.ParseIP("8.8.8.8")}})
			t.Cleanup(monkeyPatch().Reset)
			var got bool
			if got = c.IsLocalNotAllowedToSend(tt.addr, tt.asciiDomain); got != tt.want {
				t.Errorf("IsLocalNotAllowedToSend() = %v, want %v", got, tt.want)
			}
			if got2 := c.IsLocalNotAllowedToSend(tt.addr, tt.asciiDomain); got != got2 {
				t.Errorf("IsLocalNotAllowedToSend() 2nd call = %v, want %v", got2, got)
			}
		})
	}
}

func TestNewCache(t *testing.T) {
	conf := Configuration{}
	got := NewCache(&conf)
	if got.conf != &conf {
		t.Errorf("conf not set")
	}
	if got.cache == nil {
		t.Errorf("cache not set")
	}
}
