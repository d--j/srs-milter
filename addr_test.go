package srsmilter

import (
	"net/mail"
	"reflect"
	"testing"
)

func Test_split(t *testing.T) {
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

func Test_parseAddressList(t *testing.T) {
	tests := []struct {
		name     string
		in       string
		wantList []*mail.Address
		wantErr  bool
	}{
		{"empty", "", nil, false},
		{"one", "root@localhost", []*mail.Address{{Address: "root@localhost"}}, false},
		{"padded", "  ,,,root@localhost\n,", []*mail.Address{{Address: "root@localhost"}}, false},
		{"two", "  ,,,root@localhost\n,second@example.com\r\r\n", []*mail.Address{{Address: "root@localhost"}, {Address: "second@example.com"}}, false},
		{"broken", "  ,,,root@localhost\n,second@example.com\rbroken\r\n", nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotList, err := parseAddressList(tt.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAddressList() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotList, tt.wantList) {
				t.Errorf("parseAddressList() gotList = %v, want %v", gotList, tt.wantList)
			}
		})
	}
}
