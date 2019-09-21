package main

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
