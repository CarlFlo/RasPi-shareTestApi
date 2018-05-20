package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

// Struct
type Help struct {
	Success  bool   `json:"success"`
	Commands string `json: "commands"`
}

type DebugInfo struct {
	Instances  int    `json:"instances"`
	Served     int    `json:"served"`
	Updated    int    `json:"updated"`
	Removed    int    `json:"removed"`
	DebugCalls int    `json:"debugCalls"`
	Alloc      uint64 `json:"alloc_kB"`
	TotalAlloc uint64 `json:"totalAlloc_kB"`
	Sys        uint64 `json:"sys_kB"`
	IpGlobal   string `json:"ipGlobal"`
	IpLocal    string `json:"ipLocal"`
	Port       int    `json:"port"`
}

type Instance struct {
	Success bool   `json:"success"`
	ID      string `json:"id"`
	Text    string `json:"text"`
}

type GlobalIPCall struct {
	IP string `json:"ip"`
}

var DATA = make(map[string]Instance)

var PORT = 8888
var GLOBAL_IP string
var LOCAL_IP string

var DISPLAY_GUI = true
var MAX_GEN_SIZE = 1000
var DELETE_TIME int = 3600 // 1h
var TOTAL_INSTANCES_SERVED int = 0
var TOTAL_INSTANCES_UPDATED int = 0
var TOTAL_INSTANCES_REMOVED int = 0
var TOTAL_DEBUG_CALLS int = 0

// Skapa instans
func createInstance(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	var newI Instance
	params := mux.Vars(r)

	_ = json.NewDecoder(r.Body).Decode(&newI)

	newI.Success = true
	newI.ID = genID()
	newI.Text = params["text"]
	DATA[newI.ID] = newI

	json.NewEncoder(w).Encode(newI)
	go autoRemove(newI.ID)
	updateScreen()
}

// genererar ett unikt ID
func genID() string {

	for {
		id := strconv.Itoa(rand.Intn(MAX_GEN_SIZE))
		if _, ok := DATA[id]; !ok {
			return id
		}
	}
}

func autoRemove(keyId string) {
	time.Sleep(time.Duration(DELETE_TIME*1000) * time.Millisecond)
	fmt.Println("REMOVED ID: " + keyId)
	delete(DATA, keyId)
	TOTAL_INSTANCES_REMOVED++
	updateScreen()
}

// hämta instans från id
func getInstance(w http.ResponseWriter, r *http.Request) {

	TOTAL_INSTANCES_SERVED++
	updateScreen()

	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)

	// Checkar om datan finns
	if val, ok := DATA[params["id"]]; ok {
		json.NewEncoder(w).Encode(val)
	} else {
		json.NewEncoder(w).Encode(Instance{
			Success: false,
			ID:      params["id"]})
	}
}

// Uppdatera datan på en instans
func updateInstance(w http.ResponseWriter, r *http.Request) {

	TOTAL_INSTANCES_UPDATED++
	updateScreen()

	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	if _, ok := DATA[params["id"]]; ok {
		tmp := DATA[params["id"]]
		tmp.Text = params["text"]
		DATA[params["id"]] = tmp
		json.NewEncoder(w).Encode(DATA[params["id"]])
	} else {
		json.NewEncoder(w).Encode(Instance{
			Success: false,
			ID:      params["id"]})
	}
}

func debug(w http.ResponseWriter, r *http.Request) {

	TOTAL_DEBUG_CALLS++

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	json.NewEncoder(w).Encode(DebugInfo{
		Instances:  len(DATA),
		Served:     TOTAL_INSTANCES_SERVED,
		Updated:    TOTAL_INSTANCES_UPDATED,
		Removed:    TOTAL_INSTANCES_REMOVED,
		DebugCalls: TOTAL_DEBUG_CALLS,
		Alloc:      m.Alloc / 1024,
		TotalAlloc: m.TotalAlloc / 1024,
		Sys:        m.Sys / 1024,
		IpGlobal:   GLOBAL_IP,
		IpLocal:    LOCAL_IP,
		Port:       PORT})

	updateScreen()
}

func help(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Help{Success: true, Commands: "/api/help /api/new/{text} /api/get/{id} /api/update/{id}/{text} /api/debug"})
}

func getOutboundIP() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	LOCAL_IP = localAddr.IP.String()
}

func getGlobalIP() {

	url := "https://api.ipify.org?format=json"

	res, err := http.Get(url)
	if err != nil {
		GLOBAL_IP = "err"
		return
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		GLOBAL_IP = "err"
		return
	}

	var s = new(GlobalIPCall)

	errParse := json.Unmarshal(body, &s)
	if errParse != nil {
		GLOBAL_IP = "err"
		return
	}

	GLOBAL_IP = s.IP
}

func clear() {
	cmd := exec.Command("cmd", "/c", "cls")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func updateScreen() {

	if !DISPLAY_GUI {
		return
	}

	clear()
	fmt.Print("Global: ", GLOBAL_IP, ":", PORT, "\n")
	fmt.Print("Local: ", LOCAL_IP, ":", PORT, "\n")
	fmt.Print("\nInstances: ", len(DATA), "\nServed: ", TOTAL_INSTANCES_SERVED, "\nUpdated: ", TOTAL_INSTANCES_UPDATED, "\nRemoved: ", TOTAL_INSTANCES_REMOVED, "\nDebug Calls: ", TOTAL_DEBUG_CALLS, "\n")

	fmt.Print("\n\nCommands:\n/api/help\n/api/new/{text}\n/api/get/{id}\n/api/update/{id}/{text}\n/api/debug\n")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Print("\n\nMemory usage:\nAlloc: ", m.Alloc/1024, " kB\nTotalAlloc: ", m.TotalAlloc/1024, " kB\nSys: ", m.Sys/1024, " kB\n")
}

func main() {
	getGlobalIP()
	getOutboundIP()
	updateScreen()

	allowedHeaders := handlers.AllowedHeaders([]string{"X-Requested-With"})
	allowedOrigins := handlers.AllowedOrigins([]string{"*"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "DELETE", "OPTIONS"})

	// Skapar router
	r := mux.NewRouter()

	// Handlers
	r.HandleFunc("/api/help", help).Methods("GET")
	r.HandleFunc("/api/new/{text}", createInstance).Methods("GET")
	r.HandleFunc("/api/get/{id}", getInstance).Methods("GET")
	r.HandleFunc("/api/update/{id}/{text}", updateInstance).Methods("GET")
	r.HandleFunc("/api/debug", debug).Methods("GET")

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", PORT), handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(r)))
	//log.Fatal(http.ListenAndServe(":"+PORT, handlers.CORS(allowedHeaders, allowedOrigins, allowedMethods)(r)))	// HEROKU
}
