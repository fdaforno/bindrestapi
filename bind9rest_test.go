package main

import (
	"net/http"
	"testing"
)

func TestCheckDnsEntry(t *testing.T) {
	type args struct {
		rec  string
		data string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"valid A record", args{"A", "8.8.8.8"}, true},
		{"invalid A record", args{"A", "10.A.12.13"}, false},
		{"valid CNAME record", args{"CNAME", "abc.domain.com"}, true},
		{"invalid CNAME record", args{"CNAME", "AAAAAaaaaa."}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CheckDnsEntry(tt.args.rec, tt.args.data); got != tt.want {
				t.Errorf("CheckDnsEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

//TODO mock command executor
func TestCreateDNSEntry(t *testing.T) {
	type args struct {
		w   http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}