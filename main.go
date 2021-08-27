package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/Kubiuks/Alife_web/web_lib"
)

var tpl = template.Must(template.ParseFiles("index.html"))

type Agent struct {
	ID   int
	X, Y float64
}

type All_agents struct {
	Agents   []Agent
	Num      int
	Finished bool
}

type Parameters struct {
	NumAgents                    int
	World, BondedAgents, DSImode string
}

var chGrid chan []web_lib.Agent
var chComm chan string

func receive_agents_from_sim() All_agents {
	var data All_agents
	agents := <-chGrid
	data.Finished = false
	if agents == nil {
		data.Finished = true
	}
	data.Agents = make([]Agent, len(agents))
	data.Num = len(agents)
	for i := 0; i < len(agents); i++ {
		data.Agents[i] = Agent{agents[i].ID(), agents[i].X(), agents[i].Y()}
	}
	return data
}

func comm_simulation() {
	chComm <- "stop"
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	buf := &bytes.Buffer{}
	err := tpl.Execute(buf, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buf.WriteTo(w)
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Serve Agents positions taken from the simulation
		data := receive_agents_from_sim()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
		return
	case http.MethodPost:
		// Start a new Simulation
		var params Parameters
		err := json.NewDecoder(r.Body).Decode(&params)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		chGrid = make(chan []web_lib.Agent)
		chComm = make(chan string)
		go runSim(params.NumAgents, params.World, params.BondedAgents, params.DSImode, chGrid, chComm)
		data := receive_agents_from_sim()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(data)
		return
	case http.MethodPut:
		// Send command to the Simulation
		// TODO get actual data from frontend
		comm_simulation()
		return
	default:
		// Give an error message.
	}
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8443"
	}

	mux := http.NewServeMux()

	fs := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fs))

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/simulation", agentsHandler)
	http.ListenAndServe(":"+port, mux)
}
