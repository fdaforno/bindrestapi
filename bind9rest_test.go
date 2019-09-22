package main

import (
	"net"
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
		{"valid PTR record", args{"PTR", "8.8.8.8"}, true},
		{"invalid PTR record", args{"A", "10.A.12.13"}, false},
		{"valid CNAME record", args{"CNAME", "abc.domain.com"}, true},
		{"invalid CNAME record", args{"CNAME", "AAAAAaaaaa."}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DnsEntryExists(tt.args.rec, tt.args.data); got != tt.want {
				t.Errorf("DnsEntryExists() = %v, want %v", got, tt.want)
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

func TestReverseIPAddress(t *testing.T) {
	type args struct {
		ip net.IP
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{"valid IP", args{net.ParseIP("1.2.3.4")}, "4.3.2.1", false},
		{"invalid IP", args{net.ParseIP("1.A.3.4")}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReverseIPAddress(tt.args.ip)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReverseIPAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReverseIPAddress() got = %v, want %v", got, tt.want)
			}
		})
	}
}