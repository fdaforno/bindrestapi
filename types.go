package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const (
	ErrMalformedJSON        = "error parsing JSON input into struct of type %T: %v"
	ErrMissingFields        = "missing or empty data required for record %T"
	ErrMalformedIPv4Address = "malformed IP address for record %T: %s"
	ErrNotFound             = "record %T not found: %s"
	ErrOverlap              = "record %T already configured: %s"
)

type Configuration struct {
	App AppInfo
}

type AppInfo struct {
	PortListen      string
	LogsPath        string
	NsupdateKeyPath string
}

type RecordA struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Commit bool   `json:"Commit"`
}

type RecordPTR struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Commit bool   `json:"NoWrite"`
}

type RecordCNAME struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Commit bool   `json:"Commit"`
}

/*type RecordSRV struct {
	service  string
	ip       string
	priority string
	weight   string
	port     string
	target   string
}*/
type Response struct {
	Info  string
	Error string
}

type DnsRecord interface {
	Create(r io.Reader) (string, error)
	Delete(r io.Reader) (string, error)
	Read(r io.Reader) (string, error)
	valid() bool
}

func (a *RecordA) Read(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(a); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *a, err)
	}
	if DnsEntryExists("A", a.IP) {
		return "FOUND", nil
	}
	return "NOT FOUND", nil
}

func (a *RecordA) Delete(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(a); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *a, err)
	}
	if a.valid() {
		if DnsEntryExists("A", a.IP) {
			//create the dnsExec command
			if a.Commit {
				//check if the ip is valid BEFORE calling the dns command
				rec, err := ReverseIPAddress(net.ParseIP(a.IP))
				if err != nil {
					return "", fmt.Errorf(ErrMalformedIPv4Address, *a, a.IP)
				}
				dnsExec("update delete " + a.Name + " 300 A " + a.IP + "\n\r")
				dnsExec("update delete " + rec + ".in-addr.arpa. 300 PTR " + a.Name + "\n\r")
				return "record has been successfully deleted", nil
			}
			return "record is eligible for deletion, but was not deleted", nil
		}
		return "record not found", fmt.Errorf(ErrNotFound, *a, a.IP)
	}
	return "", fmt.Errorf(ErrMissingFields, *a)
}

func (a *RecordA) Create(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(a); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *a, err)
	}
	if a.valid() {
		if !DnsEntryExists("A", a.IP) {
			//create the dnsExec command
			if a.Commit {
				//check if the ip is valid BEFORE calling the dns command
				rec, err := ReverseIPAddress(net.ParseIP(a.IP))
				if err != nil {
					return "", fmt.Errorf(ErrMalformedIPv4Address, *a, a.IP)
				}
				dnsExec("update add " + a.Name + " 300 A " + a.IP + "\n\r")
				dnsExec("update add " + rec + ".in-addr.arpa. 300 PTR " + a.Name + "\n\r")
				return "record has been successfully created", nil
			}
			return "record is eligible for creation, but was not created", nil
		}
		return "record already present", fmt.Errorf(ErrOverlap, *a, a.IP)
	}
	return "", fmt.Errorf(ErrMissingFields, *a)

}

func (a *RecordA) valid() bool {
	return a.IP != "" && a.Name != ""
}

func (c *RecordCNAME) Read(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(c); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *c, err)
	}
	if DnsEntryExists("CNAME", c.Name) {
		return "FOUND", nil
	}
	return "NOT FOUND", nil
}

func (c *RecordCNAME) Delete(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(c); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *c, err)
	}
	if c.valid() {
		if DnsEntryExists("CNAME", c.Name) {
			//create the dnsExec command
			if c.Commit {
				dnsExec("update delete " + c.Name + " 300 CNAME " + c.Target + "\n\r")
				return "record has been successfully deleted", nil
			}
			return "record is eligible for deletion, but was not deleted", nil
		}
		return "record not found", fmt.Errorf(ErrNotFound, *c, c.Name)
	}
	return "", fmt.Errorf(ErrMissingFields, *c)
}

func (c *RecordCNAME) Create(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(c); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *c, err)
	}
	if c.valid() {
		if !DnsEntryExists("CNAME", c.Name) {
			//create the dnsExec command
			if c.Commit {
				dnsExec("update add " + c.Name + " 300 CNAME " + c.Target + "\n\r")
				return "record has been successfully created", nil
			}
			return "record is eligible for creation, but was not created", nil
		}
		return "record already present", fmt.Errorf(ErrOverlap, *c, c.Name)
	}
	return "", fmt.Errorf(ErrMissingFields, *c)
}

func (c *RecordCNAME) valid() bool {
	return c.Target != "" && c.Name != ""
}

func (p *RecordPTR) Read(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(p); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *p, err)
	}
	if DnsEntryExists("CNAME", p.IP) {
		return "FOUND", nil
	}
	return "NOT FOUND", nil
}

func (p *RecordPTR) Delete(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(p); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *p, err)
	}
	if p.valid() {
		if DnsEntryExists("PTR", p.Name) {
			if p.Commit {
				dnsExec("update delete " + p.IP + ".in-addr.arpa. 300 PTR " + p.Name + "\n\r")
				return "record has been successfully deleted", nil
			}
			return "record is eligible for deletion, but was not deleted", nil
		}
		return "record not found", fmt.Errorf(ErrNotFound, *p, p.Name)
	}
	return "", fmt.Errorf(ErrMissingFields, *p)
}

func (p *RecordPTR) Create(r io.Reader) (string, error) {
	if err := json.NewDecoder(r).Decode(p); err != nil {
		return "", fmt.Errorf(ErrMalformedJSON, *p, err)
	}
	if p.valid() {
		if !DnsEntryExists("CNAME", p.Name) {
			//create the dnsExec command
			if p.Commit {
				dnsExec("update add " + p.IP + ".in-addr.arpa. 300 PTR " + p.Name + "\n\r")
				return "record has been successfully created", nil
			}
			return "record is eligible for creation, but was not created", nil
		}
		return "record already present", fmt.Errorf(ErrOverlap, *p, p.Name)
	}
	return "", fmt.Errorf(ErrMissingFields, *p)
}

func (p *RecordPTR) valid() bool {
	return p.IP != "" && p.Name != ""
}
