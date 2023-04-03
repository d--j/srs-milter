package srsmilter

import (
	"context"
	"time"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/inconshreveable/log15"
)

func Filter(_ context.Context, trx mailfilter.Trx, config *Configuration, cache *Cache) (mailfilter.Decision, error) {
	startTime := time.Now()
	didSomething := false
	fromIsSrs := trx.MailFrom().AsciiDomain() == config.SrsDomain.String() && looksLikeSrs(trx.MailFrom().Local())
	hasRemoteTo := false
	hasDkim := trx.Headers().Value("Dkim-Signature") != ""
	toReplacements := make(map[string]string)

	logger := Log.New(log15.Ctx{"qid": trx.QueueId(), "user": trx.MailFrom().AuthenticatedUser()})
	logger.Debug("start", log15.Ctx{"ofrom": trx.MailFrom().Addr})

	// change any rcpt to that is pointing to our SRS domain back to the real address
	for _, to := range trx.RcptTos() {
		if to.AsciiDomain() != config.SrsDomain.String() || !looksLikeSrs(to.Local()) {
			logger.Debug("to is not one of our SRS addresses", log15.Ctx{"to": to.Addr})
			continue
		}
		didSomething = true
		rewrittenTo, err := ReverseSrs(to.Addr, config)
		if err != nil {
			logger.Error("error while generating reverse SRS address", log15.Ctx{"oto": to.Addr, "to": rewrittenTo, "err": err})
		} else {
			logger.Info("reverse SRS", log15.Ctx{"oto": to.Addr, "to": rewrittenTo})
			toReplacements[to.Addr] = rewrittenTo
			trx.AddRcptTo(rewrittenTo, "")
			trx.DelRcptTo(to.Addr)
		}
	}

	// Change the return path when it's not already one of my SRS and the mail goes to another MTA
	// … but only when there is an SPF record for the return path that prevents me from sending without SRS
	if !fromIsSrs && trx.MailFrom().Addr != "" {
		for _, to := range trx.RcptTos() {
			for _, t := range config.ResolveForward(to) {
				if t.Addr != "" && !config.IsLocalDomain(t.AsciiDomain()) {
					hasRemoteTo = true
					logger.Info("to is remote", "to", t.Addr, "transport", t.Transport())
					break
				}
				logger.Debug("to is not remote", "to", t.Addr, "transport", t.Transport())
			}
		}
		if hasRemoteTo && cache.IsLocalNotAllowedToSend(trx.MailFrom().Addr, trx.MailFrom().AsciiDomain()) {
			didSomething = true
			srsAddress, err := ForwardSrs(trx.MailFrom().Addr, config)
			if err != nil {
				logger.Error("error while generating SRS address", log15.Ctx{"ofrom": trx.MailFrom().Addr, "from": srsAddress, "err": err})
			} else {
				logger.Info("SRS", log15.Ctx{"ofrom": trx.MailFrom().Addr, "from": srsAddress})
				// Sendmail does not like getting ESMTP args, so we always send empty ESMTP args
				trx.ChangeMailFrom(srsAddress, "")
			}
		}
	}

	// fix up To:, Cc: and Bcc: headers -- but only if there are no DKIM-Signatures that we might break
	// (we assume that the To header is secured by the DKIM signature – almost all do this)
	if len(toReplacements) > 0 && !hasDkim {
		fields := trx.Headers().Fields()
		for fields.Next() {
			switch fields.CanonicalKey() {
			case "To", "Cc", "Bcc":
			default:
				continue
			}
			addresses, err := fields.AddressList()
			if err != nil {
				logger.Warn("error parsing address list, skipping", log15.Ctx{"key": fields.Key(), "value": fields.Value(), "err": err})
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
				logger.Info("fixing MIME header", log15.Ctx{"key": fields.Key(), "ovalue": fields.Value(), "addresses": addresses})
			} else {
				logger.Debug("nothing to do", log15.Ctx{"key": fields.Key(), "value": fields.Value(), "addresses": addresses})
			}
		}
	} else if len(toReplacements) > 0 && hasDkim {
		logger.Info("did not touch MIME headers because of DKIM")
	}

	if didSomething {
		logger.Info("done", "dur", time.Now().Sub(startTime))
	} else {
		logger.Debug("done", "dur", time.Now().Sub(startTime))
	}

	return mailfilter.Accept, nil
}
