package srsmilter

import (
	"database/sql"
	"fmt"
	"net"
	"net/mail"
	"strings"

	"github.com/d--j/go-milter/mailfilter/addr"
	"github.com/inconshreveable/log15"
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
	DbDriver       string
	DbDSN          string
	DbForwardQuery string
	db             *sql.DB
	localDomainMap map[string]bool
}

func (c *Configuration) Setup() error {
	c.localDomainMap = make(map[string]bool)
	for _, d := range c.LocalDomains {
		c.localDomainMap[d.String()] = true
	}
	if c.DbDriver != "" && c.DbDSN != "" && c.DbForwardQuery != "" {
		db, err := sql.Open(c.DbDriver, c.DbDSN)
		if err != nil {
			return err
		}
		err = db.Ping()
		if err != nil {
			return err
		}
		c.db = db
	}
	return nil
}

func (c *Configuration) IsLocalDomain(asciiDomain string) bool {
	if c.localDomainMap[asciiDomain] {
		return true
	}
	return false
}

func emailWithoutExtension(local string, asciiDomain string) string {
	plus := strings.IndexByte(local, '+')
	if plus < 1 {
		return fmt.Sprintf("%s@%s", local, asciiDomain)
	}
	return fmt.Sprintf("%s@%s", local[:plus-1], asciiDomain)
}

func (c *Configuration) ResolveForward(email *addr.RcptTo) (emails []*addr.RcptTo) {
	if c.db == nil {
		return []*addr.RcptTo{email}
	}
	rows, err := c.db.Query(c.DbForwardQuery, emailWithoutExtension(email.Local(), email.AsciiDomain()))
	if err != nil {
		Log.Warn("query error looking up forwards", "email", email.Addr, "err", err)
		return []*addr.RcptTo{email}
	}
	defer rows.Close()
	for rows.Next() {
		dest := ""
		if err = rows.Scan(&dest); err != nil {
			Log.Warn("scan error looking up forwards", "email", email.Addr, "err", err)
			return []*addr.RcptTo{email}
		}
		addresses, err := mail.ParseAddressList(dest)
		if err != nil {
			Log.Warn("parse error looking up forwards", "email", email.Addr, "err", err)
			return []*addr.RcptTo{email}
		}
		for _, a := range addresses {
			emails = append(emails, addr.NewRcptTo(a.Address, "", email.Transport()))
		}
	}
	if len(emails) > 0 {
		return emails
	}
	return []*addr.RcptTo{email}
}

var Log = log15.New()

func init() {
	Log.SetHandler(log15.DiscardHandler())
}
