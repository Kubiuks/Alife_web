package main

import (
	"errors"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Kubiuks/Alife_web/web_lib"
	"github.com/Kubiuks/Alife_web/web_model"
)

func runSim(numAg int, wD, bA, DSIm string, chGrid chan []web_lib.Agent, chComm chan string) {
	start := time.Now()
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------VARIABLES TESTED IN THE EXPERIMENT--------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	// variables tested in the experiment
	worldDynamics := wD
	numberOfAgents := numAg
	bondedAgents := bonds(bA)
	DSImode := DSIm

	// for now always Neutral, so 0.5 for every agent
	cortisolThresholdCondition := "Neutral"
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	//----------------------------------------------------------------------------------------------------------------------
	rand.Seed(time.Now().UnixNano())

	iterations := 15000

	// world setup
	w, h := 99, 99
	visionLength := 20
	// vision angle is both to the right and left so actually twice this amount
	// vision in degrees and must be smaller than 90 (overall smaller than 180)
	visionAngle := 40

	a := web_lib.NewSimulation()
	grid2D := web_model.NewWorld(w, h, numberOfAgents, visionLength, visionAngle)
	a.SetWorld(grid2D)

	// channel for communication with the Engine (ABM)
	a.SetComm(chComm)

	//check if correct mode
	errDSI := checkDSImode(DSImode)
	if errDSI != nil {
		log.Fatal(errDSI)
	}

	// initialise agents from 1 to numOfAgents
	for i := 1; i < numberOfAgents+1; i++ {
		x, y := randomFloat(float64(w)), randomFloat(float64(h))
		addAgent(x, y, i, i, numberOfAgents, a, grid2D, false, cortisolThresholdCondition, DSImode)
	}

	// set up bonds between agents
	errBond := initialiseBonds(bondedAgents, numberOfAgents, a)
	if errBond != nil {
		log.Fatal(errBond)
	}

	// pick world settings
	setupWorld(a, grid2D, worldDynamics)

	a.LimitIterations(iterations)

	// reporting function, does something each iteration
	// in this case updates the UI
	a.SetReportFunc(func(a *web_lib.ABM) {
		chGrid <- a.Agents()
	})

	a.StartSimulation()
	close(chGrid)
	close(chComm)

	elapsed := time.Since(start)
	log.Printf("runtime: %s", elapsed)
}

//______________________________________________________________________________________________________________________
//______________________________________________________________________________________________________________________
//______________________________________________________________________________________________________________________
//______________________________________________________________________________________________________________________
//______________________________________________________________________________________________________________________

func addAgent(x, y float64, id, rank, numOfAgents int, a *web_lib.ABM, grid2D *web_model.Grid,
	trail bool, CortisolThresholdCondition, DSImode string) {
	cell, err := web_model.NewAgent(a, id, rank, numOfAgents, x, y, trail, CortisolThresholdCondition, DSImode)
	if err != nil {
		log.Fatal(err)
	}
	a.AddAgent(cell)
	grid2D.SetCell(cell.X(), cell.Y(), cell)
}

func addFood(x, y float64, a *web_lib.ABM, grid2D *web_model.Grid) {
	cell, err := web_model.NewFood(a, x, y)
	if err != nil {
		log.Fatal(err)
	}
	a.AddAgent(cell)
	grid2D.SetCell(cell.X(), cell.Y(), cell)
}

func setupWorld(a *web_lib.ABM, grid2D *web_model.Grid, condition string) {
	err := grid2D.SetWorldDynamics(condition)
	if err != nil {
		log.Fatal(err)
	}
	switch condition {
	case "Static":
		addFood(9, 9, a, grid2D)
		addFood(89, 89, a, grid2D)
		addFood(9, 89, a, grid2D)
		addFood(89, 9, a, grid2D)
	case "Seasonal":
		addFood(9, 9, a, grid2D)
		addFood(89, 89, a, grid2D)
		addFood(9, 89, a, grid2D)
		addFood(89, 9, a, grid2D)
	case "Extreme":
		addFood(9, 9, a, grid2D)
		addFood(89, 89, a, grid2D)
		addFood(9, 89, a, grid2D)
		addFood(89, 9, a, grid2D)
	}
}

func initialiseBonds(bondedAgents []int, numberOfAgents int, a *web_lib.ABM) error {
	for i := 0; i < len(bondedAgents); i++ {
		if bondedAgents[i] < 1 || bondedAgents[i] > numberOfAgents {
			return errors.New("invalid agent id. Agents id is an int and must be from range {1:numOfAgents}")
		}
		for j := 0; j < len(bondedAgents); j++ {
			if i != j {
				if bondedAgents[i] == bondedAgents[j] {
					return errors.New("agent bond duplicate; agent cannot bond with itself")
				}
			}
		}
	}
	for i := 0; i < len(bondedAgents); i++ {
		for _, agent := range a.Agents() {
			if agent.ID() == bondedAgents[i] {
				var bonds []int
				for j := 0; j < len(bondedAgents); j++ {
					if i != j {
						bonds = append(bonds, bondedAgents[j])
					}
				}
				agent.(*web_model.Agent).SetBonds(bonds)
			}
		}
	}
	return nil
}

func bonds(arg string) []int {

	temp := strings.Replace(arg, "[", "", -1)
	temp2 := strings.Replace(temp, "]", "", -1)

	t := strings.Split(temp2, ",")

	if t[0] == "" {
		return nil
	}

	var t2 []int

	for _, i := range t {
		j, err := strconv.Atoi(i)
		if err != nil {
			log.Fatal(err)
		}
		t2 = append(t2, j)
	}
	return t2
}

func checkDSImode(DSImode string) error {
	if DSImode == "Fixed" || DSImode == "Variable" {
		return nil
	}
	return errors.New("DSImode must be one of: Fixed, Variable")
}

// needed to make sure it's never 0
func randomFloat(max float64) float64 {
	var res float64
	for {
		res = rand.Float64()
		if res != 0 {
			break
		}
	}
	return res * max
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
