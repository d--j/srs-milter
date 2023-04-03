package main

import (
	"github.com/go-logfmt/logfmt"
	"github.com/inconshreveable/log15"
)

func LogfmtFormatWithTime() log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		common := []interface{}{r.KeyNames.Time, r.Time, r.KeyNames.Lvl, r.Lvl, r.KeyNames.Msg, r.Msg}
		b, _ := logfmt.MarshalKeyvals(append(common, r.Ctx...)...)
		b = append(b, '\n')
		return b
	})
}

func LogfmtFormatWithoutTime() log15.Format {
	return log15.FormatFunc(func(r *log15.Record) []byte {
		common := []interface{}{r.KeyNames.Lvl, r.Lvl, r.KeyNames.Msg, r.Msg}
		b, _ := logfmt.MarshalKeyvals(append(common, r.Ctx...)...)
		b = append(b, '\n')
		return b
	})
}
