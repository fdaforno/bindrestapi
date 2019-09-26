package main

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"net/http/httptest"
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

func TestReadDNSEntry(t *testing.T) {
	w := httptest.NewRecorder()
	type args struct {
		w   http.ResponseWriter
		req *http.Request
	}
	tests := []struct {
		name   string
		args   args
		hasErr bool
	}{
		{"valid record A, present", args{w, httptest.NewRequest(http.MethodGet,
			"/A",
			bytes.NewBufferString(`{"name":"abc","ip":"8.8.8.8","commit":true}`))}, false},
		{"valid record A, not present", args{w, httptest.NewRequest(http.MethodGet,
			"/A",
			bytes.NewBufferString(`{"name":"abc","ip":"192.168.0.0","commit":true}`))}, false},
		{"valid record CNAME, present", args{w, httptest.NewRequest(http.MethodGet,
			"/CNAME",
			bytes.NewBufferString(`{"name":"www.facebook.com","target":"star-mini.c10r.facebook.com","commit":true}`))}, false},
		{"valid record CNAME, not present", args{w, httptest.NewRequest(http.MethodGet,
			"/CNAME",
			bytes.NewBufferString(`{"name":"www.blahblah.blah","target":"aaaaaaaa","commit":true}`))}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ReadDNSEntry(w, tt.args.req)
			if !tt.hasErr {
				assert.Contains(t, string(w.Body.Bytes()), "FOUND")
			} else {
				assert.Contains(t, string(w.Body.Bytes()), "parsing JSON input")
			}
		})
	}
}
