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
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Commit bool   `json:"NoWrite"`
}
type RecordPTR struct {
	Name   string `json:"name"`
	IP     string `json:"ip"`
	Commit bool   `json:"NoWrite"`
}
type RecordCNAME struct {
	Name   string `json:"name"`
	Target string `json:"target"`
	Commit bool   `json:"NoWrite"`
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

func nsupdate(command string) {
	// NS commnad //
	// server 127.0.0.1
	// update add 131.182.120.10.in-addr.arpa. 300 PTR kubef5ingress.qa-pci.bravofly.intra
	// send
	subProcess := exec.Command("nsupdate", "-k", "/etc/rndc.key") //Just for testing, replace with your subProcess

	stdin, err := subProcess.StdinPipe()
	if err != nil {
		fmt.Println(err)
	}
	defer stdin.Close() // the doc says subProcess.Wait will close it, but I'm not sure, so I kept this line

	subProcess.Stdout = os.Stdout
	subProcess.Stderr = os.Stderr

	if err = subProcess.Start(); err != nil { //Use start, not run
		log.Error("nsupdate ->", err)
	}

	io.WriteString(stdin, "server 127.0.0.1\n\r")
	io.WriteString(stdin, command)
	io.WriteString(stdin, "show\n\r")
	io.WriteString(stdin, "send\n\r")
	io.WriteString(stdin, "quit\n\r")
	subProcess.Wait()
	duration := time.Duration(3) * time.Second
	time.Sleep(duration)
}
func CheckDnsEntry(t string, data string) bool {

	if t == "A" {
		a, err := net.LookupAddr(data)
		if err != nil {
			log.Info("CheckDnsEntry -> " + err.Error())
			return false
		}
		if len(a) <= 0 {
			log.Info("CheckDnsEntry -> CNAME " + data + " not exist")
			return false
		} else {
			log.Info("CheckDnsEntry -> " + data + " is A of " + a[0])
			return true
		}
	} else if t == "CNAME" {

		cname, err := net.LookupCNAME(data)
		if err != nil {
			log.Info("CheckDnsEntry -> " + err.Error())
			return false
		}
		if len(cname) <= 0 {
			log.Info("CheckDnsEntry -> CNAME " + data + " not exist")
			return false
		} else {
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
	//var jrecordSRV RecordSRV
	var jrecordPTR RecordPTR
	if req.RequestURI == "/A" {
		//decode the json data
		err := decoder.Decode(&jrecordA)
		if err != nil {
			log.Error("CreateDNSEntry A -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordA.IP != "" && jrecordA.Name != "" {
				//check if the ip is valid
				if net.ParseIP(jrecordA.IP) != nil {
					//check if already exist
					if CheckDnsEntry("A", jrecordA.IP) == false {
						//create the nsupdate command
						nsupdatecommand := "update add " + jrecordA.Name + " 300 A " + jrecordA.IP + "\n\r"

						if jrecordA.Commit {
							log.Info("CreateDNSEntry -> Command=" + nsupdatecommand)
							nsupdate(nsupdatecommand)
						} else {
							log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
						}
						if CheckDnsEntry("A", jrecordA.IP) == true {
							log.Info("CreateDNSEntry A -> Done")
							returnjson("ok", false, w)
						} else {
							returnjson("Error updating entry dns", true, w)
						}
					} else {
						returnjson("Alredy exist", true, w)
					}
				} else {
					log.Error("CreateDNSEntry -> Ip=" + jrecordA.IP + " is not valid")
					returnjson("Error ip malformed", true, w)
				}
			} else {
				log.Error("CreateDNSEntry -> json field empty")
				returnjson("Json field empty", true, w)
			}
		} /*-----/A ----*/
	} else if req.RequestURI == "/CNAME" {
		//decode the json data
		err := decoder.Decode(&jrecordCNAME)
		if err != nil {
			log.Error("CreateDNSEntry CNAME -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
				// check if CNAME already exist
				if CheckDnsEntry("CNAME", jrecordCNAME.Name) == false {
					//create the nsupdate command
					nsupdatecommand := "update add " + jrecordCNAME.Name + " 300 CNAME " + jrecordCNAME.Target + "\n\r"
					if jrecordCNAME.Commit {
						log.Info("ExecuteNsUpdate -> nscommand=" + nsupdatecommand)
						nsupdate(nsupdatecommand)
					} else {
						log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
					}
					//verify if the dns entry are load
					if CheckDnsEntry("CNAME", jrecordCNAME.Name) == true {
						log.Info("CreateDNSEntry CNAME -> Done")
						returnjson("ok", false, w)
					} else {
						log.Error("CreateDNSEntry CNAME -> Error updating entry dns!!!!")
						returnjson("Error updating entry dns", true, w)
					}
				} else {
					returnjson("Alredy exist", false, w)
				}
			} else {
				returnjson("Some field was empty", true, w)
			}
		} /*-----/CNAME ----*/
	} else if req.RequestURI == "/PTR" {
		//decode the json data
		err := decoder.Decode(&jrecordPTR)
		if err != nil {
			log.Error("CreateDNSEntry PTR -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordPTR.IP != "" && jrecordPTR.Name != "" {
				//check if the ip is valid
				if net.ParseIP(jrecordPTR.IP) != nil {
					//check if already exist
					if CheckDnsEntry("PTR", jrecordPTR.IP) == false {
						//create the nsupdate command
						nsupdatecommand := "update add " + jrecordPTR.IP + ".in-addr.arpa. 300 PTR " + jrecordPTR.Name + "\n\r"

						if jrecordA.Commit {
							log.Info("CreateDNSEntry -> Command=" + nsupdatecommand)
							nsupdate(nsupdatecommand)
						} else {
							log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
						}
						if CheckDnsEntry("PTR", jrecordPTR.IP) == true {
							log.Info("CreateDNSEntry PTR -> Done")
							returnjson("ok", false, w)
						} else {
							returnjson("Error updating entry dns", true, w)
						}
					} else {
						returnjson("Alredy exist", true, w)
					}
				} else {
					log.Error("CreateDNSEntry -> Ip=" + jrecordPTR.IP + " is not valid")
					returnjson("Error ip malformed", true, w)
				}
			} else {
				log.Error("CreateDNSEntry -> json field empty")
				returnjson("Json field empty", true, w)
			}
		}
	} /*-----/SRV ----*/
}

func DeleteDNSEntry(w http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var jrecordA RecordA
	var jrecordCNAME RecordCNAME
	//var jrecordSRV RecordSRV
	var jrecordPTR RecordPTR
	if req.RequestURI == "/A" {
		//decode the json data
		err := decoder.Decode(&jrecordA)
		if err != nil {
			log.Error("DeleteDNSEntry A -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordA.IP != "" && jrecordA.Name != "" {
				//check if the ip is valid
				if net.ParseIP(jrecordA.IP) != nil {
					//check if already exist
					if CheckDnsEntry("A", jrecordA.IP) == true {
						//create the nsupdate command
						nsupdatecommand := "update delete " + jrecordA.Name + " 300 A " + jrecordA.IP + "\n\r"

						if jrecordA.Commit {
							log.Info("DeleteDNSEntry -> Command=" + nsupdatecommand)
							nsupdate(nsupdatecommand)
						} else {
							log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
						}
						if CheckDnsEntry("A", jrecordA.IP) == true {
							log.Error("CreateDNSEntry A -> Error updating entry dns!!!!")
							returnjson("Error updating entry dns", true, w)
						} else {
							log.Info("DeleteDNSEntry A -> Done")
							returnjson("ok", false, w)
						}
					} else {
						returnjson("Alredy exist", true, w)
					}
				} else {
					log.Error("CreateDNSEntry -> Ip=" + jrecordA.IP + " is not valid")
					returnjson("Error ip malformed", true, w)
				}
			} else {
				log.Error("CreateDNSEntry -> json field empty")
				returnjson("Json field empty", true, w)
			}
		} /*-----/A ----*/
	} else if req.RequestURI == "/CNAME" {
		//decode the json data
		err := decoder.Decode(&jrecordCNAME)
		if err != nil {
			log.Error("DeleteDNSEntry CNAME -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordCNAME.Target != "" && jrecordCNAME.Name != "" {
				// check if CNAME already exist
				if CheckDnsEntry("CNAME", jrecordCNAME.Name) == false {
					//create the nsupdate command
					nsupdatecommand := "update delete " + jrecordCNAME.Name + " 300 CNAME " + jrecordCNAME.Target + "\n\r"
					if jrecordCNAME.Commit {
						log.Info("ExecuteNsUpdate -> nscommand=" + nsupdatecommand)
						nsupdate(nsupdatecommand)
					} else {
						log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
					}
					//verify if the dns entry are load
					if CheckDnsEntry("CNAME", jrecordCNAME.Name) == true {
						log.Error("CreateDNSEntry CNAME -> Error updating entry dns!!!!")
						returnjson("Error updating entry dns", true, w)
					} else {
						log.Info("DeleteDNSEntry CNAME -> Done")
						returnjson("ok", false, w)
					}
				} else {
					returnjson("Alredy exist", false, w)
				}
			} else {
				returnjson("Some field was empty", true, w)
			}
		} /*-----/CNAME ----*/
	} else if req.RequestURI == "/PTR" {
		//decode the json data
		err := decoder.Decode(&jrecordPTR)
		if err != nil {
			log.Error("DeleteDNSEntry PTR -> Error parsing input json")
			returnjson("Error parsing input json", true, w)
		} else {
			//check if all fileds are filled
			if jrecordPTR.IP != "" && jrecordPTR.Name != "" {
				//check if the ip is valid
				if net.ParseIP(jrecordPTR.IP) != nil {
					//check if already exist
					if CheckDnsEntry("A", jrecordPTR.IP) == true {
						//create the nsupdate command
						nsupdatecommand := "update delete " + jrecordPTR.IP + ".in-addr.arpa. 300 PTR " + jrecordPTR.Name + "\n\r"

						if jrecordA.Commit {
							log.Info("DeleteDNSEntry -> Command=" + nsupdatecommand)
							nsupdate(nsupdatecommand)
						} else {
							log.Info("TestNsUpdate -> nscommand=" + nsupdatecommand)
						}
						if CheckDnsEntry("A", jrecordPTR.IP) == true {
							log.Error("CreateDNSEntry PTR -> Error updating entry dns!!!!")
							returnjson("Error updating entry dns", true, w)
						} else {
							log.Info("DeleteDNSEntry PTR -> Done")
							returnjson("ok", false, w)
						}
					} else {
						returnjson("Alredy exist", true, w)
					}
				} else {
					log.Error("CreateDNSEntry -> Ip=" + jrecordPTR.IP + " is not valid")
					returnjson("Error ip malformed", true, w)
				}
			} else {
				log.Error("CreateDNSEntry -> json field empty")
				returnjson("Json field empty", true, w)
			}
		}
	} /*-----/SRV ----*/
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

	log.SetOutput(f)
	log.SetLevel(log.DebugLevel)

	router.HandleFunc("/", PrintUsage).Methods("GET")
	router.HandleFunc("/A", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/CNAME", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/PTR", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/SRV", CreateDNSEntry).Methods("POST")
	router.HandleFunc("/A", DeleteDNSEntry).Methods("DELETE")
	router.HandleFunc("/CNAME", DeleteDNSEntry).Methods("DELETE")
	router.HandleFunc("/PTR", DeleteDNSEntry).Methods("DELETE")

	log.Info("--------------------------------")
	log.Info("        BIND REST STARTED       ")
	log.Info("--------------------------------")
	log.Info("Git Commit Hash: %s\n", GitHash)
	log.Info("UTC Build Time: %s\n", BuildStamp)
	log.Info("Port: " + conf.App.PortListen)
	daemon.SdNotify(false, "READY=1")
	log.Fatal(http.ListenAndServe(conf.App.PortListen, router))
}
