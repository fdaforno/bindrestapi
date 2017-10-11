package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"time"

	"net/http"
	"os"
	"os/exec"

	"github.com/BurntSushi/toml"
	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-systemd/daemon"
	"github.com/gorilla/mux"
)

var (
	BuildStamp = "Nothing Provided."
	GitHash    = "Nothing Provided."
)

type configfile struct {
	App AppInfo
}
type AppInfo struct {
	PortListen      string
	LogsPath        string
	NsupdateKeyPath string
}
type RecordA struct {
	Name  string `json:"name"`
	Class string `json:"class"`
	IP    string `json:"ip"`
}
type RecordCNAME struct {
	Name   string `json:"name"`
	Class  string `json:"class"`
	Target string `json:"target"`
}
type RecordSRV struct {
	service  string
	class    string
	priority string
	weight   string
	port     string
	target   string
}

type Response struct {
	Info  string
	Error string
}

func nsupdate(command string) {
	// NS commnad //
	// server 127.0.0.1
	// update add 131.182.120.10.in-addr.arpa. 300 PTR kubef5ingress.qa-pci.bravofly.intra
	// send
	subProcess := exec.Command("nsupdate", "-k", "/etc/rndc.key") //Just for testing, replace with your subProcess

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		fmt.Println(err) //replace with logger, or anything you want
	}
	defer stdin.Close() // the doc says subProcess.Wait will close it, but I'm not sure, so I kept this line

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil { //Use start, not run
		log.Error("nsupdate ->", err) //replace with logger, or anything you want
	}

	io.WriteString(stdin, "server 127.0.0.1\n\r")
	io.WriteString(stdin, command)
	io.WriteString(stdin, "show\n\r")
	io.WriteString(stdin, "send\n\r")
	io.WriteString(stdin, "quit\n\r")
	subProcess.Wait()
	duration := time.Duration(2) * time.Second
	time.Sleep(duration)
}
func CheckDnsEntry(t string, data string) bool {

	if t == "A" {
		a, err := net.LookupAddr(data)
		if err != nil {
			return false
		}
		if len(a) <= 0 {
			return false
		} else {
			return true
		}
	} else if t == "CNAME" {

		cname, err := net.LookupCNAME(data)
		if err != nil {
			fmt.Println("CheckDnsEntry -> Cname not exist ")
			log.Info("CheckDnsEntry -> Cname not exist ")
			return false
		}
		if len(cname) <= 0 {
			fmt.Println("CheckDnsEntry -> CNAME " + data + " not exist")
			log.Info("CheckDnsEntry -> CNAME " + data + " not exist")
			return false
		} else {
			fmt.Println("CheckDnsEntry -> " + data + " is a CNAME  of " + cname)
			log.Info("CheckDnsEntry -> " + data + " is a CNAME  of " + cname)
			return true
		}
	} else if t == "/SRV" {

	}
	return true
}
func PrintUsage(w http.ResponseWriter, req *http.Request) {
}

func CreateDNSEntry(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var jrecordA RecordA
	var jrecordCNAME RecordCNAME
	var jrecordSRV RecordSRV

	if req.RequestURI == "/A" {

		err := decoder.Decode(&jrecordA)

		if err != nil {
			log.WithFields(log.Fields{"Note": &jrecordA}).Error("CreateDNSEntry A -> Error parsing input jsong")
			w.WriteHeader(500)
		} else {
			if jrecordA.Class != "" && jrecordA.IP != "" && jrecordA.Name != "" {

				nsupdatecommand := "update add " + jrecordA.Name + " 300 " + jrecordA.Class + " " + jrecordA.IP + "\n\r"
				log.WithFields(log.Fields{"Command": nsupdatecommand}).Info("CreateDNSEntry -> ")

				//nsupdate(nsupdatecommand)
				w.WriteHeader(200)
			} else {
				w.WriteHeader(500)
			}
		}
	} else if req.RequestURI == "/CNAME" {
		err := decoder.Decode(&jrecordCNAME)

		if err != nil {
			log.Error("CreateDNSEntry CNAME -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {

			if jrecordCNAME.Class != "" && jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
				/*----- CHECK IF CNAME ALREADY EXIST ----*/

				if CheckDnsEntry(jrecordCNAME.Class, jrecordCNAME.Name) == false {
					/*----- CREATE CNAME ----*/

					nsupdatecommand := "update add " + jrecordCNAME.Name + " 300 " + jrecordCNAME.Class + " " + jrecordCNAME.Target + "\n\r"
					fmt.Println(nsupdatecommand)
					log.Info("CreateDNSEntry -> nscommand=" + nsupdatecommand)

					nsupdate(nsupdatecommand)

					/*----- CHECK IF CNAME EXIST ----*/
					if CheckDnsEntry(jrecordCNAME.Class, jrecordCNAME.Name) == false {
						log.Error("qualcosa e andato storto")
						returnjson("Error updating entry dns", true, w)

					} else {
						returnjson("ok", false, w)
					}
				} else {
					/*----- CNAME ALREADY EXISTT ----*/

					returnjson("Alredy exist", false, w)
				}

			} else {
				returnjson("Some field was empty", true, w)
			}
		}

	} else if req.RequestURI == "/SRV" {
		fmt.Println("trying to parse as record SRV")
		err := decoder.Decode(&jrecordSRV)
		if err != nil {
			w.Write([]byte("Error"))
			w.WriteHeader(500)
		} else {
			if jrecordSRV.service != "" {
				fmt.Println(jrecordSRV.port)
				w.Write([]byte("OK"))
			} else {
				w.WriteHeader(500)
			}
		}
	}
}

func DeleteDNSEntry(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var jrecordA RecordA
	var jrecordCNAME RecordCNAME
	var jrecordSRV RecordSRV
	if req.RequestURI == "/A" {

		err := decoder.Decode(&jrecordA)

		if err != nil {
			log.WithFields(log.Fields{"Note": &jrecordA}).Error("DeleteDNSEntry A -> Error parsing input jsong")
			w.WriteHeader(500)
		} else {
			if jrecordA.Class != "" && jrecordA.IP != "" && jrecordA.Name != "" {

				nsupdatecommand := "update add " + jrecordA.Name + " 300 " + jrecordA.Class + " " + jrecordA.IP + "\n\r"
				log.WithFields(log.Fields{"Command": nsupdatecommand}).Info("CreateDNSEntry -> ")

				//nsupdate(nsupdatecommand)
				w.WriteHeader(200)
			} else {
				w.WriteHeader(500)
			}
		}
	} else if req.RequestURI == "/CNAME" {
		err := decoder.Decode(&jrecordCNAME)

		if err != nil {
			returnjson("Error parsing input json", true, w)
		} else {

			if jrecordCNAME.Class != "" && jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
				/*----- CHECK IF CNAME ALREADY EXIST ----*/
				if CheckDnsEntry(jrecordCNAME.Class, jrecordCNAME.Name) == true {
					/*----- DELETE CNAME ----*/
					nsupdatecommand := "update delete " + jrecordCNAME.Name + " " + jrecordCNAME.Class + " " + jrecordCNAME.Target + "\n\r"
					fmt.Println(nsupdatecommand)
					log.Info("DeleteDNSEntry -> nscommand=" + nsupdatecommand)

					nsupdate(nsupdatecommand)

					/*----- CHECK IF CNAME EXIST ----*/
					if CheckDnsEntry(jrecordCNAME.Class, jrecordCNAME.Name) == false {
						returnjson("ok", false, w)
					} else {
						returnjson("Error deleting entry dns", true, w)
					}
				} else {
					/*----- CNAME ALREADY EXISTT ----*/
					returnjson("Entry Not exist", true, w)
				}

			} else {
				returnjson("Some field was empty", true, w)
			}
		}

	} else if req.RequestURI == "/SRV" {
		fmt.Println("trying to parse as record SRV")
		err := decoder.Decode(&jrecordSRV)
		if err != nil {
			w.Write([]byte("Error"))
			w.WriteHeader(500)
		} else {
			if jrecordSRV.service != "" {
				fmt.Println(jrecordSRV.port)
				w.Write([]byte("OK"))
			} else {
				w.WriteHeader(500)
			}
		}
	}
}
func returnjson(message string, iserror bool, w http.ResponseWriter) {
	w.Header().Add("Content-Type", "application/json")
	r := Response{}

	if iserror {
		w.WriteHeader(http.StatusBadRequest)
		r.Error = message
		rjson, _ := json.Marshal(r)
		w.Write(rjson)
	} else {
		w.WriteHeader(http.StatusAccepted)
		r.Info = message
		rjson, _ := json.Marshal(r)
		w.Write(rjson)
	}
}
func main() {

	// copy bind9rest.service in /lib/systemd/system/
	// systectl enable bind9rest.service
	// systectl start bind9rest.service

	// A and CNAME {"name":"prova", "class":"A", "ip":"10.10.10.2"}
	// SRV {"service":"prova", "class":"A", "priority":"1", "weight":"10","port":"1111","target":"sticazzi"}

	/* --- READ THE CONFIGURUATION FILE --- */
	var conf configfile
	if _, err := toml.DecodeFile("config.toml", &conf); err != nil {
		// handle error
		fmt.Printf("error opening file: %v", err)
	}

	router := mux.NewRouter()
	/* ---- INIT LOG ---- */
	// open a file
	f, err := os.OpenFile(conf.App.LogsPath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	defer f.Close()
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)

	router.HandleFunc("/", PrintUsage).Methods("GET")
	router.HandleFunc("/A", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/CNAME", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/SRV", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/CNAME", DeleteDNSEntry).Methods("DELETE")

	log.Info("--------------------------------")
	log.Info("        BIND REST STARTED       ")
	log.Info("--------------------------------")
	log.Info("Git Commit Hash: %s\n", GitHash)
	log.Info("UTC Build Time: %s\n", BuildStamp)
	log.Info("Port: " + conf.App.PortListen)
	daemon.SdNotify(false, "READY=1")
	log.Fatal(http.ListenAndServe(conf.App.PortListen, router))
}
