package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/coreos/go-systemd/daemon"
	"github.com/gorilla/mux"
	"io"
	"net"
	"strings"
	"time"

	"net/http"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

var (
	BuildStamp string
	GitHash    string
)

func main() {
	// copy bind9rest.service in /lib/systemd/system/
	// systemctl enable bind9rest.service
	// systemctl start bind9rest.service
	configPath := flag.String("f", "config.toml", "path to the configuration file")
	flag.Parse()

	/* --- READ THE CONFIGURATION FILE --- */
	var conf Configuration
	if _, err := toml.DecodeFile(*configPath, &conf); err != nil {
		// handle error, exit because has no sense to continue if config file is not there
		log.Fatalf("error opening configuration file at path %s: %v", *configPath, err)
	}

	/* ---- INIT LOG ---- */
	// defaults
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	// open a file
	f, err := os.OpenFile(conf.App.LogsPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	if err != nil {
		log.Warnf("error opening destination log file %s: %v; falling back to StdOut", conf.App.LogsPath, err)
	} else {
		log.SetOutput(f)
		defer f.Close()
	}

	/* ---- HTTP ROUTES ---- */
	router := mux.NewRouter()
	router.HandleFunc("/", PrintUsage).Methods("GET")
	router.HandleFunc("/A", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/CNAME", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/PTR", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/SRV", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/A", DeleteDNSEntry).Methods("DELETE")
	router.HandleFunc("/CNAME", DeleteDNSEntry).Methods("DELETE")
	router.HandleFunc("/PTR", DeleteDNSEntry).Methods("DELETE")
	router.HandleFunc("/CNAME", ReadDNSEntry).Methods("GET")
	router.HandleFunc("/A", ReadDNSEntry).Methods("GET")
	router.HandleFunc("/PTR", ReadDNSEntry).Methods("GET")

	log.Info("--------------------------------")
	log.Info("        BIND REST STARTED       ")
	log.Info("--------------------------------")
	log.Infof("Git Commit Hash: %s", GitHash)
	log.Infof("UTC Build Time: %s", BuildStamp)
	log.Info("Port: " + conf.App.PortListen)
	daemon.SdNotify(false, "READY=1")
	log.Fatal(http.ListenAndServe(conf.App.PortListen, router))
}

func dnsExec(command string) {
	// NS command
	//
	// server 127.0.0.1
	// update add 123.123.12.3.in-addr.arpa. 300 PTR something.zone.bravofly.intra
	// send
	subProcess := exec.Command("/usr/bin/dnsExec", "-k", "/etc/rndc.key")

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdin.Close() // the doc says subProcess.Wait will close it, but I'm not sure, so I kept this line

	stdout, err := subProcess.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdout.Close()
	stderr, err := subProcess.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stderr.Close()
	//subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil { //Use start, not run
		log.Errorf("dnsExec command error: %v", err)
		return
	}

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	io.WriteString(stdin, "server 127.0.0.1\n\r")
	io.WriteString(stdin, command)

	io.WriteString(stdin, "send\n\r")

	io.WriteString(stdin, "quit\n\r")
	subProcess.Wait()
	time.Sleep(time.Duration(2) * time.Second)
}

func ReverseIPAddress(ip net.IP) (string, error) {
	if ip.To4() != nil {
		// split into slice by dot .
		addressSlice := strings.Split(ip.String(), ".")
		reverseSlice := []string{}

		for i := range addressSlice {
			octet := addressSlice[len(addressSlice)-1-i]
			reverseSlice = append(reverseSlice, octet)
		}

		return strings.Join(reverseSlice, "."), nil

	}
	return "", fmt.Errorf("invalid IPv4 address: %s", ip.String())
}

func DnsEntryExists(rec, data string) bool {
	switch rec {
	case "A", "PTR":
		a, err := net.LookupAddr(data)
		if err != nil {
			log.Infof("DnsEntryExists -> %v", err)
			return false
		}
		if len(a) <= 0 {
			log.Infof("DnsEntryExists -> %s %s does not have a reverse lookup", rec, data)
			return false
		}
		log.Infof("DnsEntryExists -> %s is A of %s", data, a[0])
		return true

	case "CNAME":
		cname, err := net.LookupCNAME(data)
		if err != nil {
			log.Errorf("DnsEntryExists -> %v", err)
			return false
		}
		if len(cname) <= 0 {
			log.Infof("DnsEntryExists -> CNAME %s does not exist", data)
			return false
		}
		log.Infof("DnsEntryExists -> %s is CNAME of %s", data, cname)
		return true

	case "/SRV":
		return true
	default:
		return false
	}
}

func PrintUsage(w http.ResponseWriter, req *http.Request) {}

func CreateDNSEntry(w http.ResponseWriter, req *http.Request) {
	var record DNSRecord
	switch req.RequestURI {
	case "/A":
		var recordA RecordA
		record = &recordA
		break
	case "/CNAME":
		var recordCNAME RecordCNAME
		record = &recordCNAME
		break
	case "/PTR":
		var recordPTR RecordPTR
		record = &recordPTR
		break

	default:
		returnJSON(ErrUnsupportedRecType, true, w)
		return
	}
	//polymorphism FTW
	out, err := record.Create(req.Body)
	if err != nil {
		msg := fmt.Errorf("could not create record: %v", err)
		log.Errorf("CreateDNSEntry -> %v", msg)
		returnJSON(msg.Error(), true, w)
		return
	}
	returnJSON(out, false, w)
	return
}

func DeleteDNSEntry(w http.ResponseWriter, req *http.Request) {
	var record DNSRecord
	switch req.RequestURI {
	case "/A":
		var recordA RecordA
		record = &recordA
		break
	case "/CNAME":
		var recordCNAME RecordCNAME
		record = &recordCNAME
		break
	case "/PTR":
		var recordPTR RecordPTR
		record = &recordPTR
		break

	default:
		returnJSON(ErrUnsupportedRecType, true, w)
		return
	}

	out, err := record.Delete(req.Body)
	if err != nil {
		msg := fmt.Errorf("could not delete record: %v", err)
		log.Errorf("DeleteDNSEntry -> %v", msg)
		returnJSON(msg.Error(), true, w)
		return
	}
	returnJSON(out, false, w)
	return
}

func ReadDNSEntry(w http.ResponseWriter, req *http.Request) {
	var record DNSRecord
	switch req.RequestURI {
	// modified with new interface definition
	case "/A":
		var recordA RecordA
		record = &recordA
		break

	case "/CNAME":
		var recordCNAME RecordCNAME
		record = &recordCNAME
		break

	case "/PTR":
		var recordPTR RecordPTR
		record = &recordPTR

	default:
		returnJSON(ErrUnsupportedRecType, true, w)
		return
	}
	out, err := record.Read(req.Body)
	if err != nil {
		log.Error("ReadDNSEntry A -> Error parsing JSON input")
		returnJSON(out, true, w)
	}
	returnJSON(out, false, w)
}

func returnJSON(message string, isErr bool, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	r := Response{}
	if isErr {
		w.WriteHeader(http.StatusBadRequest)
		r.Error = message
		json.NewEncoder(w).Encode(&r)
		return
	}
	w.WriteHeader(http.StatusAccepted)
	r.Info = message
	json.NewEncoder(w).Encode(&r)
	return
}
