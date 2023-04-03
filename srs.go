package srsmilter

import (
	"errors"
	"strings"

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
		if err != nil && err != srs.ErrHashInvalid {
			return "", err
		}
		if err == nil {
			return addr, nil
		}
	}
	return "", errors.New("no SRS key found or all tried keys failed")
}

func looksLikeSrs(local string) bool {
	return strings.HasPrefix(local, "SRS0=") || strings.HasPrefix(local, "SRS1=") || strings.HasPrefix(local, "srs0=") || strings.HasPrefix(local, "srs1=")
}
