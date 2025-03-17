package srsmilter

import (
	"testing"
)

func TestForwardSrs(t *testing.T) {
	c2 := &Configuration{
		SrsDomain: "srs.example.com",
		SrsKeys:   []string{},
	}
	c3 := &Configuration{
		SrsDomain: "srs.example.com",
		SrsKeys:   []string{"secret-key", "another"},
	}
	type args struct {
		addr   string
		config *Configuration
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"no key", args{"abc", c2}, "", true},
		{"not-an-email", args{"hello - at - example.com", c3}, "", true},
		{"my-srs-key-rotation", args{"someone@example.net", c3}, "SRS0=R9Ph=46=example.net=someone@srs.example.com", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(monkeyPatch().Reset)
			got, err := ForwardSrs(tt.args.addr, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ForwardSrs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ForwardSrs() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReverseSrs(t *testing.T) {
	c1 := &Configuration{
		SrsDomain: "srs.example.com",
		SrsKeys:   []string{"one", "two"},
	}
	c2 := &Configuration{
		SrsDomain: "srs.example.com",
		SrsKeys:   []string{},
	}
	c3 := &Configuration{
		SrsDomain: "srs.example.com",
		SrsKeys:   []string{"one", "secret-key"},
	}
	type args struct {
		srsAddress string
		config     *Configuration
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"no key", args{"abc", c2}, "", true},
		{"not-my-srs", args{"SRS0=R9Ph=46=example.net=someone@srs.example.net", c1}, "", true},
		{"not-an-address", args{"hello - at - example.com", c1}, "", true},
		{"my-srs-key-rotation", args{"SRS0=R9Ph=46=example.net=someone@srs.example.com", c3}, "someone@example.net", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(monkeyPatch().Reset)
			got, err := ReverseSrs(tt.args.srsAddress, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReverseSrs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReverseSrs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
