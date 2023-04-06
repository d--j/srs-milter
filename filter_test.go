package srsmilter

import (
	"context"
	"net"
	"reflect"
	"testing"

	"github.com/d--j/go-milter/mailfilter"
	"github.com/d--j/go-milter/mailfilter/addr"
	"github.com/d--j/go-milter/mailfilter/testtrx"
	"github.com/emersion/go-message/mail"
)

func TestFilter(t *testing.T) {
	t.Cleanup(monkeyPatch().Reset)
	conf := &Configuration{
		SrsDomain:    "srs.example.com",
		LocalDomains: []Domain{ToDomain("example.com")},
		SrsKeys:      []string{"secret-key"},
		LocalIps:     []net.IP{net.ParseIP("8.8.8.8")},
		LogLevel:     3,
	}
	conf.Setup()
	cache := NewCache(conf)
	newTrx := func() *testtrx.Trx {
		return (&testtrx.Trx{}).
			SetMTA(mailfilter.MTA{
				Version: "Postfix 2.3.0",
				FQDN:    "mx.example.com",
				Daemon:  "smtpd",
			}).
			SetConnect(mailfilter.Connect{
				Host:   "localhost",
				Family: "tcp",
				Port:   25,
				Addr:   "127.0.0.1",
				IfName: "lo",
				IfAddr: "127.0.0.1",
			}).
			SetHelo(mailfilter.Helo{
				Name:        "localhost",
				TlsVersion:  "",
				Cipher:      "",
				CipherBits:  "",
				CertSubject: "",
				CertIssuer:  "",
			}).
			SetMailFrom(addr.NewMailFrom("somebody@example.com", "", "smtp", "", "")).
			SetRcptTosList("to@example.com").
			SetHeadersRaw([]byte("Subject: test\r\n\r\n")).
			SetBodyBytes([]byte("body"))
	}
	type args struct {
		trx    *testtrx.Trx
		config *Configuration
		cache  *Cache
	}
	tests := []struct {
		name              string
		args              args
		want              mailfilter.Decision
		wantModifications []testtrx.Modification
		wantErr           bool
	}{
		{"no-op", args{newTrx(), conf, cache}, mailfilter.Accept, nil, false},
		{"forward-not-local", args{
			newTrx().
				SetMailFrom(addr.NewMailFrom("not-local@example.net", "", "smtp", "", "")).
				SetRcptTosList("someone@example.net"),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{{Kind: testtrx.ChangeFrom, Addr: "SRS0=+5us=46=example.net=not-local@srs.example.com"}}, false},
		{"forward-not-local-no-spf", args{
			newTrx().
				SetMailFrom(addr.NewMailFrom("not-local-no-spf@example.org", "", "smtp", "", "")).
				SetRcptTosList("someone@example.net"),
			conf, cache,
		}, mailfilter.Accept, nil, false},
		{"forward-not-local-srs1", args{
			newTrx().
				SetMailFrom(addr.NewMailFrom("SRS0=ABCD=46=example.org=not-local-srs1@example.net", "", "smtp", "", "")).
				SetRcptTosList("someone@example.net"),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{{Kind: testtrx.ChangeFrom, Addr: "SRS1=TWks=example.net==ABCD=46=example.org=not-local-srs1@srs.example.com"}}, false},
		{"forward-bogus-email", args{
			newTrx().
				SetMailFrom(addr.NewMailFrom("(not-local@example.net", "", "smtp", "", "")).
				SetRcptTosList("someone@example.net"),
			conf, cache,
		}, mailfilter.Accept, nil, false},
		{"reverse-local", args{
			newTrx().
				SetRcptTosList("local@example.com"),
			conf, cache,
		}, mailfilter.Accept, nil, false},
		{"reverse-my-srs", args{
			newTrx().
				SetRcptTosList("SRS0=PNjA=46=example.net=my-srs@srs.example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=PNjA=46=example.net=my-srs@srs.example.com>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{
			{Kind: testtrx.DelRcptTo, Addr: "SRS0=PNjA=46=example.net=my-srs@srs.example.com"},
			{Kind: testtrx.AddRcptTo, Addr: "my-srs@example.net"},
			{Kind: testtrx.ChangeHeader, Index: 1, Name: "To", Value: " \"Someone\" <my-srs@example.net>"},
		}, false},
		{"reverse-my-srs-multi", args{
			newTrx().
				SetRcptTosList("SRS0=PNjA=46=example.net=my-srs@srs.example.com", "another@example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=PNjA=46=example.net=my-srs@srs.example.com>, Another <another@example.com>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{
			{Kind: testtrx.DelRcptTo, Addr: "SRS0=PNjA=46=example.net=my-srs@srs.example.com"},
			{Kind: testtrx.AddRcptTo, Addr: "my-srs@example.net"},
			{Kind: testtrx.ChangeHeader, Index: 1, Name: "To", Value: " \"Someone\" <my-srs@example.net>,\r\n \"Another\" <another@example.com>"},
		}, false},
		{"reverse-my-srs-dkim", args{
			newTrx().
				SetRcptTosList("SRS0=PNjA=46=example.net=my-srs@srs.example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=PNjA=46=example.net=my-srs@srs.example.com>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\nDKIM-Signature: bogus\n\n")),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{
			{Kind: testtrx.DelRcptTo, Addr: "SRS0=PNjA=46=example.net=my-srs@srs.example.com"},
			{Kind: testtrx.AddRcptTo, Addr: "my-srs@example.net"},
		}, false},
		{"reverse-my-srs-err", args{
			newTrx().
				SetRcptTosList("SRS0=XXXX=46=example.net=my-srs@srs.example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=XXXX=46=example.net=my-srs@srs.example.com>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, nil, false},
		{"reverse-my-srs-header-err", args{
			newTrx().
				SetRcptTosList("SRS0=PNjA=46=example.net=my-srs@srs.example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=PNjA=46=example.net=my-srs@srs.example.com>,,(broken\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{
			{Kind: testtrx.DelRcptTo, Addr: "SRS0=PNjA=46=example.net=my-srs@srs.example.com"},
			{Kind: testtrx.AddRcptTo, Addr: "my-srs@example.net"},
		}, false},
		{"reverse-my-srs-cc", args{
			newTrx().
				SetRcptTosList("SRS0=PNjA=46=example.net=my-srs@srs.example.com").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=PNjA=46=example.net=my-srs@srs.example.com>\nCc: <boss@example.com>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, []testtrx.Modification{
			{Kind: testtrx.DelRcptTo, Addr: "SRS0=PNjA=46=example.net=my-srs@srs.example.com"},
			{Kind: testtrx.AddRcptTo, Addr: "my-srs@example.net"},
			{Kind: testtrx.ChangeHeader, Index: 1, Name: "To", Value: " \"Someone\" <my-srs@example.net>"},
		}, false},
		{"reverse-other-srs", args{
			newTrx().
				SetRcptTosList("SRS0=R9Ph=46=example.net=other-srs@srs.example.net").
				SetHeadersRaw([]byte("From: Someone <someone@example.net>\nTo: Someone <SRS0=R9Ph=46=example.net=other-srs@srs.example.net>\nSubject: Test\nDate: Fri, 10 Mar 2023 23:29:35 +0000 (UTC)\nMessage-ID: <id@example.com>\n\n")),
			conf, cache,
		}, mailfilter.Accept, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trx := tt.args.trx
			got, err := Filter(context.Background(), trx, tt.args.config, tt.args.cache)
			if (err != nil) != tt.wantErr {
				t.Errorf("Filter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Filter() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(trx.Modifications(), tt.wantModifications) {
				t.Errorf("trx.Modifications() got = %v, want %v", trx.Modifications(), tt.wantModifications)
			}
		})
	}
}

func Test_outputAddresses(t *testing.T) {
	type args struct {
		addrs []*mail.Address
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{"one", args{[]*mail.Address{{Address: "root@localhost"}}}, "root@localhost"},
		{"nil", args{[]*mail.Address{{Address: "root@localhost"}, nil}}, "root@localhost"},
		{"two", args{[]*mail.Address{{Address: "root@localhost"}, {Address: "root@localhost"}}}, "root@localhost,root@localhost"},
		{"nil-in-middle", args{[]*mail.Address{{Address: "root@localhost"}, nil, {Address: "root@localhost"}}}, "root@localhost,root@localhost"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := outputAddresses(tt.args.addrs); got != tt.want {
				t.Errorf("outputAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}
