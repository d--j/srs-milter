package srsMilter

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/mileusna/srs"
)

func ForwardSrs(addr string, config *Configuration) (string, error) {
	if len(config.SrsKeys) == 0 {
		return "", errors.New("no SRS key found")
	}
	s := srs.SRS{
		Secret:         []byte(config.SrsKeys[0]),
		Domain:         config.SrsDomain.String(),
		FirstSeparator: "=",
	}
	srsAddress, err := s.Forward(addr)
	if err != nil {
		return "", err
	}
	return srsAddress, nil
}

func ReverseSrs(srsAddress string, config *Configuration) (string, error) {
	for _, key := range config.SrsKeys {
		s := srs.SRS{
			Secret:         []byte(key),
			Domain:         config.SrsDomain.String(),
			FirstSeparator: "=",
		}
		addr, err := s.Reverse(srsAddress)
		if err != nil && err.Error() != "Hash invalid in SRS address" {
			return "", err
		}
		if err == nil {
			return addr, nil
		}
	}
	return "", errors.New("no SRS key found or all tried keys failed")
}

func looksLikeSrs(local string) bool {
	return strings.HasPrefix(local, "SRS0=") || strings.HasPrefix(local, "SRS1=")
}

func Filter(_ context.Context, trx mailfilter.Trx, config *Configuration, cache *Cache) (mailfilter.Decision, error) {
	var startTime time.Time
	didSomething := false
	fromIsSrs := trx.MailFrom().AsciiDomain() == config.SrsDomain.String() && looksLikeSrs(trx.MailFrom().Local())
	hasRemoteTo := false
	toReplacements := make(map[string]string)

	if config.LogLevel > 0 {
		startTime = time.Now()
		if config.LogLevel > 1 {
			trx.Log("from=<%s> isSRS=%v isSRSDomain=%v isSRSLocal=%v", trx.MailFrom().Addr, fromIsSrs, trx.MailFrom().AsciiDomain() == config.SrsDomain.String(), looksLikeSrs(trx.MailFrom().Local()))
		}
	}
	// Change the return path when it's not already one of my SRS and the mail goes to another MTA
	// … but only when there is an SPF record for the return path that prevents me from sending without SRS
	if !fromIsSrs && trx.MailFrom().Addr != "" {
		for _, to := range trx.RcptTos() {
			if config.LogLevel > 2 {
				trx.Log("to=<%s>", to.Addr)
			}
			if to.Addr != "" && !config.HasLocalDomain(to.AsciiDomain()) {
				hasRemoteTo = true
				break
			}
		}
		if config.LogLevel > 1 {
			trx.Log("hasRemoteTo=%v", hasRemoteTo)
		}
		if hasRemoteTo && cache.IsLocalNotAllowedToSend(trx.MailFrom().Addr, trx.MailFrom().AsciiDomain()) {
			didSomething = true
			srsAddress, err := ForwardSrs(trx.MailFrom().Addr, config)
			if config.LogLevel > 2 {
				trx.Log("from=<%s> srsAddress=<%s> err=%v", trx.MailFrom().Addr, srsAddress, err)
			}
			if err != nil {
				trx.Log("warn=\"error while generating SRS address\" input=<%s> error=%q", trx.MailFrom().Addr, err)
			} else {
				// Sendmail does not like getting ESMTP args, so we always send empty ESMTP args
				trx.ChangeMailFrom(srsAddress, "")
			}
		}
	}

	// change any rcpt to that is pointing to our SRS domain back to the real address
	for _, to := range trx.RcptTos() {
		if config.LogLevel > 2 {
			trx.Log("to=<%s> isSRSDomain=%v isSRSLocal=%v", to.Addr, to.AsciiDomain() == config.SrsDomain.String(), looksLikeSrs(to.Local()))
		}
		if to.AsciiDomain() != config.SrsDomain.String() || !looksLikeSrs(to.Local()) {
			continue
		}
		didSomething = true
		rewrittenTo, err := ReverseSrs(to.Addr, config)
		if config.LogLevel > 2 {
			trx.Log("to=<%s> rewrittenTo=<%s> err=%v", to.Addr, rewrittenTo, err)
		}
		if err != nil {
			trx.Log("warn=\"error while generating reverse SRS address\" input=<%s> error=%q", to.Addr, err)
		} else {
			toReplacements[to.Addr] = rewrittenTo
			trx.AddRcptTo(rewrittenTo, "")
			trx.DelRcptTo(to.Addr)
		}
	}

	// fix up To:, Cc: and Bcc: headers -- but only if there are no DKIM-Signatures that we might break
	// (we assume that the To header is secured by the DKIM signature – almost all do this)
	if config.LogLevel > 2 {
		trx.Log("toReplacements=%v dkim=%v", toReplacements, trx.Headers().Value("Dkim-Signature") != "")
	}
	if len(toReplacements) > 0 && trx.Headers().Value("Dkim-Signature") == "" {
		fields := trx.Headers().Fields()
		for fields.Next() {
			switch fields.CanonicalKey() {
			case "To", "Cc", "Bcc":
			default:
				continue
			}
			addresses, err := fields.AddressList()
			if config.LogLevel > 2 {
				trx.Log("key=%s addresses=%v err=%v", fields.Key(), addresses, err)
			}
			if err != nil {
				trx.Log("warn=\"error while rewriting header\" key=%q value=%q error=%q", fields.CanonicalKey(), fields.Value(), err)
				continue
			}
			changed := false
			for _, a := range addresses {
				for search, replace := range toReplacements {
					if a.Address == search {
						a.Address = replace
						changed = true
					}
				}
			}
			if changed {
				fields.SetAddressList(addresses)
			}
		}
	}

	switch config.LogLevel {
	case 0:
	default:
		didSomething = true
		fallthrough
	case 1:
		if didSomething {
			trx.Log("from=<%s> isSRS=%v hasRemoteTo=%v toReplacements=%v duration=%s", trx.MailFrom().Addr, fromIsSrs, hasRemoteTo, toReplacements, time.Now().Sub(startTime))
		}
	}

	return mailfilter.Accept, nil
}
