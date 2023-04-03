package srsmilter

import (
	"testing"
)

func toDomainSlice(in []string) (out []Domain) {
	for _, i := range in {
		out = append(out, ToDomain(i))
	}
	return
}

func TestConfiguration_HasLocalDomain(t *testing.T) {
	tests := []struct {
		name  string
		local []Domain
		arg   string
		want  bool
	}{
		{"empty1", toDomainSlice([]string{}), "", false},
		{"empty2", toDomainSlice([]string{"example.com"}), "", false},
		{"bogus1", toDomainSlice([]string{"example.com"}), "bogus domain", false},
		{"single", toDomainSlice([]string{"example.com"}), "example.com", true},
		{"multiple1", toDomainSlice([]string{"example.com", "example.net", "example.org"}), "example.com", true},
		{"multiple2", toDomainSlice([]string{"example.com", "example.net", "example.org"}), "example.net", true},
		{"multiple3", toDomainSlice([]string{"example.com", "example.net", "example.org"}), "example.org", true},
		{"multiple4", toDomainSlice([]string{"example.com", "example.net", "example.org"}), "example.biz", false},
		{"idna1", toDomainSlice([]string{"näkkileipä.example.com", "example.net", "example.org"}), "xn--nkkileip-0zah.example.com", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Configuration{
				LocalDomains: tt.local,
			}
			c.Setup()
			if got := c.IsLocalDomain(tt.arg); got != tt.want {
				t.Errorf("IsLocalDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomain_String(t *testing.T) {
	tests := []struct {
		name string
		d    Domain
		want string
	}{
		{"bogus", ToDomain("bogus domain ü"), "bogus domain ü"},
		{"simple", ToDomain("example.net"), "example.net"},
		{"idna", ToDomain("näkkileipä.example.net"), "xn--nkkileip-0zah.example.net"},
		{"ascii", ToDomain("xn--nkkileip-0zah.example.net"), "xn--nkkileip-0zah.example.net"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDomain_Unicode(t *testing.T) {
	tests := []struct {
		name string
		d    Domain
		want string
	}{
		{"bogus", ToDomain("bogus domain ü"), "bogus domain ü"},
		{"simple", ToDomain("example.net"), "example.net"},
		{"idna", ToDomain("näkkileipä.example.net"), "näkkileipä.example.net"},
		{"ascii", ToDomain("xn--nkkileip-0zah.example.net"), "näkkileipä.example.net"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.d.Unicode(); got != tt.want {
				t.Errorf("Unicode() = %v, want %v", got, tt.want)
			}
		})
	}
}
