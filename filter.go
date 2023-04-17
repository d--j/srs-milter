package srsmilter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/go-milter/mailfilter/addr"
	"github.com/emersion/go-message/mail"
)

func Filter(_ context.Context, trx mailfilter.Trx, config *Configuration, cache *Cache) (mailfilter.Decision, error) {
	startTime := time.Now()
	fromIsSrs := trx.MailFrom().AsciiDomain() == config.SrsDomain.String() && looksLikeSrs(trx.MailFrom().Local())
	hasRemoteTo := false
	hasDkim := trx.Headers().Value("Dkim-Signature") != ""
	actions := []string(nil)

	logger := Log.New("sub", "milter", "qid", trx.QueueId(), "user", trx.MailFrom().AuthenticatedUser())
	logger.Debug("start", "ofrom", trx.MailFrom().Addr)

	// change any rcpt to that is pointing to our SRS domain back to the real address
	// (just in case that our socketmap server did not do that already)
	for _, to := range trx.RcptTos() {
		if to.AsciiDomain() != config.SrsDomain.String() || !looksLikeSrs(to.Local()) {
			logger.Debug("to is not one of our SRS addresses", "to", to.Addr)
			continue
		}
		a := to.Addr
		rewrittenTo, err := ReverseSrs(a, config)
		if err != nil {
			logger.Error("error while generating reverse SRS address", "oto", a, "to", rewrittenTo, "err", err)
		} else {
			logger.Debug("reverse SRS", "oto", a, "to", rewrittenTo)
			trx.AddRcptTo(rewrittenTo, "")
			trx.DelRcptTo(a)
			actions = append(actions, fmt.Sprintf("recipient_env:%s:%s", a, rewrittenTo))
		}
	}

	// Change the return path when it's not already one of my SRS and the mail goes to another MTA
	// … but only when there is an SPF record for the return path that prevents me from sending without SRS
	if !fromIsSrs && trx.MailFrom().Addr != "" {
		for _, to := range trx.RcptTos() {
			for _, t := range config.ResolveForward(to) {
				if t.Addr != "" && !config.IsLocalDomain(t.AsciiDomain()) {
					hasRemoteTo = true
					logger.Debug("to is remote", "to", t.Addr, "transport", t.Transport())
					break
				}
				logger.Debug("to is not remote", "to", t.Addr, "transport", t.Transport())
			}
		}
		if hasRemoteTo && cache.IsLocalNotAllowedToSend(trx.MailFrom().Addr, trx.MailFrom().AsciiDomain()) {
			a := trx.MailFrom().Addr
			srsAddress, err := ForwardSrs(a, config)
			if err != nil {
				logger.Error("error while generating SRS address", "ofrom", a, "from", srsAddress, "err", err)
			} else {
				logger.Debug("SRS", "ofrom", a, "from", srsAddress)
				// Sendmail does not like getting ESMTP args, so we always send empty ESMTP args
				trx.ChangeMailFrom(srsAddress, "")
				actions = append(actions, fmt.Sprintf("sender:%s:%s", a, srsAddress))
			}
		}
	}

	// fix up To:, Cc: and Bcc: headers -- but only if there are no DKIM-Signatures that we might break
	// (we assume that the To header is secured by the DKIM signature – almost all DKIM signers do this)
	if !hasDkim {
		fields := trx.Headers().Fields()
		for fields.Next() {
			switch fields.CanonicalKey() {
			case "To", "Cc", "Bcc":
			default:
				continue
			}
			addresses, err := fields.AddressList()
			if err != nil {
				logger.Warn("error parsing address list, skipping", "key", fields.Key(), "value", fields.Value(), "err", err)
				continue
			}
			changed := false
			for _, a := range addresses {
				to := addr.NewRcptTo(a.Address, "", "")
				if to.AsciiDomain() != config.SrsDomain.String() || !looksLikeSrs(to.Local()) {
					logger.Debug("to is not one of our SRS addresses", "to", to.Addr, "hdr", fields.Key())
					continue
				}
				rewrittenTo, err := ReverseSrs(to.Addr, config)
				if err != nil {
					logger.Error("error while generating header reverse SRS address", "oto", to.Addr, "to", rewrittenTo, "err", err)
				} else {
					logger.Debug("header reverse SRS", "oto", to.Addr, "to", rewrittenTo)
					a.Address = rewrittenTo
					changed = true
					actions = append(actions, fmt.Sprintf("recipient_hdr:%s:%s", to.Addr, rewrittenTo))
				}
			}
			if changed {
				fields.SetAddressList(addresses)
				logger.Debug("fixing MIME header", "key", fields.Key(), "ovalue", fields.Value(), "addresses", outputAddresses(addresses))
			} else {
				logger.Debug("nothing to do", "key", fields.Key(), "value", fields.Value(), "addresses", outputAddresses(addresses))
			}
		}
	} else {
		logger.Debug("did not touch MIME headers because of DKIM")
	}

	if len(actions) > 0 {
		logger.Info("done", "dur", time.Now().Sub(startTime), "actions", strings.Join(actions, ","))
	} else {
		logger.Debug("end", "dur", time.Now().Sub(startTime))
	}

	return mailfilter.Accept, nil
}

func outputAddresses(addrs []*mail.Address) string {
	b := strings.Builder{}
	for i, a := range addrs {
		if a == nil {
			continue
		}
		if i > 0 {
			b.WriteRune(',')
		}
		b.WriteString(a.Address)
	}
	return b.String()
}
