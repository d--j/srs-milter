package patches

import (
	"net"
	"strings"
	"time"

	"blitiri.com.ar/go/spf"
	"github.com/agiledragon/gomonkey/v2"
)

var ConstantDate = time.Date(2023, time.January, 1, 12, 0, 0, 0, time.UTC)

func Apply() *gomonkey.Patches {
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
