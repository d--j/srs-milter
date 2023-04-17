package srsmilter

import (
	"net/mail"
	"strings"
	"unicode"
)

func split(email string) (string, Domain) {
	at := strings.LastIndexByte(email, '@')
	if at < 0 {
		return email, ""
	}
	return email[:at], ToDomain(email[at+1:])
}

func parseAddressList(in string) (list []*mail.Address, err error) {
	parts := strings.FieldsFunc(in, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r) || r == '\r' || r == '\n'
	})
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		if a, err := mail.ParseAddress(p); err != nil {
			return nil, err
		} else {
			list = append(list, a)
		}
	}
	return list, nil
}
