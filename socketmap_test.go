package srsmilter

import (
	"net"
	"testing"
)

func TestSocketmap(t *testing.T) {
	t.Cleanup(monkeyPatch().Reset)
	conf := &Configuration{
		SrsDomain:    "srs.example.com",
		LocalDomains: []Domain{ToDomain("example.com")},
		SrsKeys:      []string{"secret-key"},
		LocalIps:     []net.IP{net.ParseIP("8.8.8.8")},
		LogLevel:     3,
	}
	conf.Setup()
	type args struct {
		lookup string
		key    string
	}
	tests := []struct {
		name       string
		args       args
		wantResult string
		wantFound  bool
		wantErr    bool
	}{
		{"wrong map", args{"wrong", ""}, "", false, false},
		{"no email", args{"decode", "something"}, "", false, false},
		{"no SRS", args{"decode", "root@localhost"}, "", false, false},
		{"SRS", args{"decode", "SRS0=PNjA=46=example.net=my-srs@srs.example.com"}, "my-srs@example.net", true, false},
		{"SRS-error", args{"decode", "SRS0=XXXX=46=example.net=my-srs@srs.example.com"}, "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotFound, err := Socketmap(conf, tt.args.lookup, tt.args.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("Socketmap() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotResult != tt.wantResult {
				t.Errorf("Socketmap() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotFound != tt.wantFound {
				t.Errorf("Socketmap() gotFound = %v, want %v", gotFound, tt.wantFound)
			}
		})
	}
}

func Test_split(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name       string
		email      string
		wantLocal  string
		wantDomain Domain
	}{
		{"empty", "", "", ""},
		{"without domain", "root", "root", ""},
		{"with domain", "root@localhost", "root", ToDomain("localhost")},
		{"with two ats", "root@crazy@localhost", "root@crazy", ToDomain("localhost")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocal, gotDomain := split(tt.email)
			if gotLocal != tt.wantLocal {
				t.Errorf("split() gotLocal = %v, want %v", gotLocal, tt.wantLocal)
			}
			if gotDomain != tt.wantDomain {
				t.Errorf("split() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
			}
		})
	}
}
