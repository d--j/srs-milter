package srsmilter

import (
	"strings"
)

func Socketmap(config *Configuration, lookup, key string) (result string, found bool, err error) {
	logger := Log.New("sub", "socketmap", "lookup", lookup, "key", key)
	if lookup != "decode" {
		logger.Debug("no decode request")
		return "", false, nil
	}
	local, domain := split(key)
	if domain.String() != config.SrsDomain.String() || !looksLikeSrs(local) {
		logger.Debug("not my SRS address")
		return "", false, nil
	}
	email, err := ReverseSrs(key, config)
	if err != nil {
		logger.Warn("error decoding", "err", err)
		return "", false, err
	}
	logger.Debug("decoded", "result", email)
	return email, true, nil
}

func split(email string) (string, Domain) {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return email, ""
	}
	return email[:at], ToDomain(email[at+1:])
}
