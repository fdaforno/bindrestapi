package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"net/http"
	"os"
	"os/exec"

	"github.com/BurntSushi/toml"
	"github.com/coreos/go-systemd/daemon"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var (
	BuildStamp = "Nothing Provided."
	GitHash    = "Nothing Provided."
)

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
	decoder := json.NewDecoder(req.Body)
	var jrecordA RecordA
	var jrecordCNAME RecordCNAME
	var jrecordPTR RecordPTR
	//var jrecordSRV RecordSRV

	switch req.RequestURI {
	case "/A":
		out, err := jrecordA.Create(req.Body)
		if err!=nil {
			msg := fmt.Errorf("could not create record: %v", err)
			log.Errorf("CreateDNSEntry -> %v", msg)
			returnJSON(msg.Error(), true, w)
			return
		}
		returnJSON(out, false, w)
		return
	//	TODO implement new interface
	case "/CNAME":
		//decode the json data
		err := decoder.Decode(&jrecordCNAME)
		if err != nil {
			log.Error("CreateDNSEntry CNAME -> Error parsing input json")
			returnJSON("Error parsing input json", true, w)
			return
		}
		//check if all fileds are filled
		if jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
			// check if CNAME already exist
			if !DnsEntryExists("CNAME", jrecordCNAME.Name) {
				//create the dnsExec command
				nsupdatecommand := "update add " + jrecordCNAME.Name + " 300 CNAME " + jrecordCNAME.Target + "\n\r"
				if jrecordCNAME.Commit {
					log.Info("ExecuteNsUpdate -> nscommand=" + nsupdatecommand)
					dnsExec(nsupdatecommand)
				} else {
					log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
				}
				/*verify if the dns entry are load
				if DnsEntryExists("CNAME", jrecordCNAME.Name) == true {
					log.Info("CreateDNSEntry CNAME -> Done")
					returnJSON("ok", false, w)
				} else {
					log.Error("CreateDNSEntry CNAME -> Error updating entry dns!!!!")
					returnJSON("Error updating entry dns", true, w)
				}*/
				returnJSON("ok", false, w)
				return
			}
			returnJSON("Already exist", true, w)
			return
		}
		returnJSON("Some field was empty", true, w)
		return
		/*-----/CNAME ----*/
	case "/PTR":
		//decode the json data
		err := decoder.Decode(&jrecordPTR)
		if err != nil {
			log.Error("CreateDNSEntry PTR -> Error parsing input json")
			returnJSON("Error parsing input json", true, w)
			return
		}
		//check if all fields are filled
		if jrecordPTR.IP != "" && jrecordPTR.Name != "" {
			//check if the ip is valid
			if net.ParseIP(jrecordPTR.IP) != nil {
				//check if already exist
				if !DnsEntryExists("PTR", jrecordPTR.IP) {
					//create the dnsExec command
					nsupdatecommand := "update add " + jrecordPTR.IP + ".in-addr.arpa. 300 PTR " + jrecordPTR.Name + "\n\r"

					if jrecordA.Commit {
						log.Info("CreateDNSEntry -> Command=" + nsupdatecommand)
						dnsExec(nsupdatecommand)
					} else {
						log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
					} /*
						if DnsEntryExists("PTR", jrecordPTR.IP) == true {
							log.Info("CreateDNSEntry PTR -> Done")
							returnJSON("ok", false, w)
						} else {
							returnJSON("Error updating entry dns", true, w)
						}*/
					returnJSON("ok", false, w)
					return
				}
				returnJSON("Already exist", true, w)
				return
			}
			log.Error("CreateDNSEntry -> Ip=" + jrecordPTR.IP + " is not valid")
			returnJSON("Error ip malformed", true, w)
			return
		}
		log.Error("CreateDNSEntry -> json field empty")
		returnJSON("Json field empty", true, w)
		return

	default:
		returnJSON("Unsupported DNS record type", true, w)
		return
	} /*-----/SRV ----*/
}

func DeleteDNSEntry(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var recordA RecordA
	var jrecordCNAME RecordCNAME
	//var jrecordSRV RecordSRV
	var jrecordPTR RecordPTR
	switch req.RequestURI {
	case "/A":
		out, err := recordA.Delete(req.Body)
		if err!=nil {
			msg := fmt.Errorf("could not delete record: %v", err)
			log.Errorf("DeleteDNSEntry -> %v", msg)
			returnJSON(msg.Error(), true, w)
			return
		}
		returnJSON(out, false, w)
		return

		/*-----/A ----*/
	case "/CNAME":
		//decode the json data
		err := decoder.Decode(&jrecordCNAME)
		if err != nil {
			log.Error("DeleteDNSEntry CNAME -> Error parsing input json")
			returnJSON("Error parsing input json", true, w)
			return
		}
		//check if all fileds are filled
		if jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
			// check if CNAME already exist
			if DnsEntryExists("CNAME", jrecordCNAME.Name) == true {
				//create the dnsExec command
				nsupdatecommand := "update delete " + jrecordCNAME.Name + " 300 CNAME " + jrecordCNAME.Target + "\n\r"
				if jrecordCNAME.Commit {
					log.Info("ExecuteNsUpdate -> nscommand=" + nsupdatecommand)
					dnsExec(nsupdatecommand)
				} else {
					log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
				}
				/*verify if the dns entry are load
				if DnsEntryExists("CNAME", jrecordCNAME.Name) == true {
					log.Error("CreateDNSEntry CNAME -> Error updating entry dns!!!!")
					returnJSON("Error updating entry dns", true, w)
				} else {
					log.Info("DeleteDNSEntry CNAME -> Done")
					returnJSON("ok", false, w)
				}*/
				returnJSON("Ok", false, w)
				return
			}
			returnJSON("Host not found ", true, w)
			return
		}
		returnJSON("Missing required fields", true, w)
		return
		/*-----/CNAME ----*/
	case "/PTR":
		//decode the json data
		err := decoder.Decode(&jrecordPTR)
		if err != nil {
			log.Error("DeleteDNSEntry PTR -> Error parsing input json")
			returnJSON("Error parsing input json", true, w)
			return
		}
		//check if all fields are filled
		if jrecordPTR.IP != "" && jrecordPTR.Name != "" {
			//check if the ip is valid
			if net.ParseIP(jrecordPTR.IP) != nil {
				//check if already exist
				if DnsEntryExists("A", jrecordPTR.IP) == true {
					//create the dnsExec command
					nsupdatecommand := "update delete " + jrecordPTR.IP + ".in-addr.arpa. 300 PTR " + jrecordPTR.Name + "\n\r"

					if recordA.Commit {
						log.Info("DeleteDNSEntry -> Command=" + nsupdatecommand)
						dnsExec(nsupdatecommand)
					} else {
						log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
					} /*
						if DnsEntryExists("PTR", jrecordPTR.IP) == true {
							log.Error("CreateDNSEntry PTR -> Error updating entry dns!!!!")
							returnJSON("Error updating entry dns", true, w)
						} else {
							log.Info("DeleteDNSEntry PTR -> Done")
							returnJSON("ok", false, w)
						}*/
					returnJSON("ok", false, w)
					return
				}
				returnJSON("Already exist", true, w)
				return
			}
			log.Error("CreateDNSEntry -> Ip=" + jrecordPTR.IP + " is not valid")
			returnJSON("Error ip malformed", true, w)
			return
		}
		log.Error("CreateDNSEntry -> json field empty")
		returnJSON("Json field empty", true, w)
		return
	}
	/*-----/SRV ----*/
}

func ReadDNSEntry(w http.ResponseWriter, req *http.Request) {
	var recordA RecordA
	var recordCNAME RecordCNAME
	var recordPTR RecordPTR

	switch req.RequestURI {
	// modified with new interface definition
	case "/A":
		out, err := recordA.Read(req.Body)
		if err!=nil {
			log.Error("ReadDNSEntry A -> Error parsing JSON input")
			returnJSON(out, true, w)
		}
		returnJSON(out, false, w)

	case "/CNAME":
		//TODO implement DNSController interface
		err := json.NewDecoder(req.Body).Decode(&recordCNAME)
		if err != nil {
			log.Error("ReadDNSEntry CNAME -> Error parsing JSON input")
			returnJSON("Error parsing input json", true, w)
		}
		if DnsEntryExists("CNAME", recordCNAME.Name) {
			returnJSON("FOUND", false, w)
			return
		}
		returnJSON("NOT FOUND", false, w)
		return

	case "/PTR":
		err := json.NewDecoder(req.Body).Decode(&recordPTR)
		if err != nil {
			log.Error("ReadDNSEntry PTR -> Error parsing JSON input")
			returnJSON("Error parsing input json", true, w)
		}
		if DnsEntryExists("PTR", recordPTR.IP) {
			returnJSON("FOUND", false, w)
			return
		}
		returnJSON("NOT FOUND", false, w)
		return
	}

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

func main() {
	// copy bind9rest.service in /lib/systemd/system/
	// systemctl enable bind9rest.service
	// systemctl start bind9rest.service

	// A and CNAME {"name":"prova", "class":"A", "ip":"10.10.10.2"}
	// SRV {"service":"prova", "class":"A", "priority":"1", "weight":"10","port":"1111","target":"sticazzi"}

	/* --- READ THE CONFIGURUATION FILE --- */
	var conf Configuration
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		// handle error, exit because has no sense to continue if config file is not there
		log.Fatalf("error opening file: %v", err)
	}

	/* ---- INIT LOG ---- */
	// open a file
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	f, err := os.OpenFile(conf.App.LogsPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
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
