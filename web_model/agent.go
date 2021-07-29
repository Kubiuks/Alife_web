package web_model

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/Kubiuks/Alife_web/web_lib"
)

// Agent implements web_lib.Agent and
// walks randomly over 2D grid.
type Agent struct {
	// web_model needed
	oxytocin                float64
	cortisol                float64
	alive                   bool
	energy                  float64
	socialness              float64
	rank                    int
	stressed                bool
	nutritionChange         float64
	socialChange            float64
	oxytocinChange          float64
	cortisolChange          float64
	adaptiveThreshold       float64
	bondPartners            []int
	DSIstrengths            []float64
	foodTimeWaiting         int
	motivation              float64
	physEffTouch            float64
	touchIntensity          float64
	tactileIntensity        float64
	DSImode                 string
	psychEffEatTogether     float64
	justEaten               bool
	sharedFoodWith          []int
	groomedWith             int
	aggressionOn            int
	eatingTogetherIntensity float64
	stepSize                float64
	tactileEat              float64

	// implementation needed
	mutex        sync.Mutex
	iteration    int
	id           int
	x, y         float64
	origx, origy float64
	grid         *Grid
	trail        bool
	direction    float64
	numOfAgents  int
}

func NewAgent(abm *web_lib.ABM, id, rank, numOfAgents int, x, y float64, trail bool, CortisolThresholdCondition, DSImode string) (*Agent, error) {
	rand.Seed(time.Now().UnixNano())
	world := abm.World()
	if world == nil {
		return nil, errors.New("agent needs a World defined to operate")
	}
	grid, ok := world.(*Grid)
	if !ok {
		return nil, errors.New("agent needs a Grid world to operate")
	}

	adaptiveThreshold, err := cortisolThreshold(rank, CortisolThresholdCondition)
	if err != nil {
		return nil, err
	}
	return &Agent{
		alive:                   true,
		energy:                  1,
		oxytocin:                1,
		cortisol:                0,
		socialness:              1,
		stressed:                false,
		nutritionChange:         0.0006,
		socialChange:            0.0005,
		oxytocinChange:          0.0005,
		cortisolChange:          0.005,
		rank:                    rank,
		adaptiveThreshold:       adaptiveThreshold,
		foodTimeWaiting:         0,
		motivation:              0,
		physEffTouch:            0.1,
		touchIntensity:          0,
		tactileIntensity:        0,
		DSImode:                 DSImode,
		psychEffEatTogether:     0.1,
		justEaten:               false,
		groomedWith:             0,
		aggressionOn:            0,
		eatingTogetherIntensity: 0,
		stepSize:                0.5,
		tactileEat:              0,

		//----------------
		id:          id,
		iteration:   0,
		origx:       x,
		origy:       y,
		x:           x,
		y:           y,
		grid:        grid,
		trail:       trail,
		direction:   rand.Float64() * 360,
		numOfAgents: numOfAgents,
	}, nil
}

func (a *Agent) Run() {
	a.iteration++

	// reset flags
	a.groomedWith = 0
	a.aggressionOn = 0

	// dont do anything if dead
	if !a.alive {
		return
	}

	a.actionSelection()

	a.updateInternals()

	// check if died in this iteration
	if a.energy <= 0 {
		a.alive = false
		a.energy = 0
		a.cortisol = 1
		a.socialness = 0
		a.oxytocin = 0
		a.grid.ClearCell(a.x, a.y, a.id)
	}
}

func (a *Agent) actionSelection() {
	foods, agents, walls := a.inVision()
	// food
	foodSalience := float64(len(foods))

	energyErr := 1 - a.energy
	eatMotivation := energyErr + (energyErr * foodSalience)
	// social
	socialErr := 1 - a.socialness
	robotSalience := 0.0
	if len(agents) > 0 {
		robotSalience = 1.0
	}
	groomMotivation := socialErr + (socialErr * robotSalience)

	a.updateCT(energyErr+socialErr, agents, foods)

	if groomMotivation > eatMotivation {
		if a.justEaten {
			if randBool() {
				a.move(mod(a.direction-90, 360))
			} else {
				a.move(mod(a.direction+90, 360))
			}
			a.justEaten = false
			a.foodTimeWaiting = 0
			a.sharedEatingFood()
			a.sharedFoodWith = nil
			a.foodTimeWaiting = 0
		} else {
			a.motivation = groomMotivation
			a.touchIntensity = a.motivation * a.physEffTouch
			a.tactileIntensity = a.touchIntensity * a.cortisol * 25
			if a.tactileIntensity < 0 {
				a.tactileIntensity = 1
			} else {
				a.tactileIntensity = math.Ceil(a.tactileIntensity)
			}
			a.pickAgent(agents, foods, walls)
		}
	} else {
		a.motivation = eatMotivation
		a.eatingTogetherIntensity = a.motivation * a.psychEffEatTogether
		a.tactileEat = a.eatingTogetherIntensity * a.cortisol
		a.findEatFood(agents, foods, walls)
	}
}

func (a *Agent) normalisedAgentVal(agent *Agent) float64 {
	rankDiff := float64((a.numOfAgents-a.rank)-(a.numOfAgents-agent.Rank())) / float64(a.numOfAgents-1)
	bond := 0.0
	DSI := 0.0
	for i, id := range a.bondPartners {
		if agent.ID() == id {
			// there is a bond
			bond = 1
			DSI = a.DSIstrengths[i]
			break
		}
	}
	normalisedAgentVal := rankDiff + (bond * DSI * a.oxytocin)
	return normalisedAgentVal
}

func (a *Agent) agentVal(agent *Agent) float64 {
	rankDiff := float64(a.rank-agent.Rank()) / float64(a.numOfAgents-1)
	bond := 0.0
	DSI := 0.0
	for i, id := range a.bondPartners {
		if agent.ID() == id {
			// there is a bond
			bond = 1
			DSI = a.DSIstrengths[i]
			break
		}
	}
	agentVal := rankDiff + (bond * DSI * a.oxytocin)
	return agentVal
}

func (a *Agent) pickAgent(agents, foods, walls []web_lib.Agent) {
	if agents == nil {
		// dont see any agent, so turn if see wall else random move
		if walls != nil {
			a.turnFromWall()
		} else {
			a.randomMove()
		}
	} else {
		// see some agents, so need to pick groom partner
		// which is the agent with highest normalisedAgentVal
		var groomPartner *Agent
		normalisedAgentVal := -1.0
		for _, temp := range agents {
			tmpnormalisedAgentVal := a.normalisedAgentVal(temp.(*Agent))
			if tmpnormalisedAgentVal >= normalisedAgentVal {
				normalisedAgentVal = tmpnormalisedAgentVal
				groomPartner = temp.(*Agent)
			}
		}
		a.groomOraggressionOrAvoid(groomPartner, foods)
	}
}

func (a *Agent) groomOraggressionOrAvoid(agent *Agent, foods []web_lib.Agent) {
	agentVal := a.agentVal(agent)
	if distance(a.x, a.y, agent.X(), agent.Y()) < 2 {
		a.socialness = a.socialness + a.tactileIntensity*0.15
		if a.stressed && a.rank > agent.Rank() && agentVal <= 1 {
			a.aggression(agent)
		} else {
			a.groom(agent)
		}
	} else {
		a.moveTo(agent)
		if agentVal < 0 {
			if len(foods) > 0 {
				a.direction = mod(a.direction-180, 360)
			} else if a.stressed {
				if randBool() {
					a.direction = mod(a.direction-(90*1.5*a.cortisol), 360)
				} else {
					a.direction = mod(a.direction+(90*1.5*a.cortisol), 360)
				}
			} else {
				if randBool() {
					a.direction = mod(a.direction-(90*a.cortisol), 360)
				} else {
					a.direction = mod(a.direction+(90*a.cortisol), 360)
				}
			}
		}
	}
}

func (a *Agent) groom(agent *Agent) {
	a.groomedWith = agent.ID()
	oxyGain := (1 - a.oxytocin) * 0.7
	a.IncreaseOT(oxyGain)
	agent.IncreaseOT(oxyGain)
	agent.ModulateCT(-1 * a.tactileIntensity * 0.2)
	if a.DSImode == "Variable" {
		a.ModulateDSI(agent.ID(), a.tactileIntensity*0.3)
		agent.ModulateDSI(a.id, a.tactileIntensity*0.3)
	}
	a.randomMove()
}

func (a *Agent) aggression(agent *Agent) {
	a.aggressionOn = agent.ID()
	a.ModulateCT(-1 * a.tactileIntensity * 0.15)
	agent.ModulateCT(a.tactileIntensity * 0.15)
	if a.DSImode == "Variable" {
		a.ModulateDSI(agent.ID(), -1*a.tactileIntensity*0.15)
		agent.ModulateDSI(a.id, -1*a.tactileIntensity*0.15)
	}
	a.randomMove()
}

func (a *Agent) avoidAgent() {
	if a.stressed {
		if randBool() {
			a.move(mod(a.direction-(90*1.5*a.cortisol), 360))
		} else {
			a.move(mod(a.direction+(90*1.5*a.cortisol), 360))

		}
	} else {
		if randBool() {
			a.move(mod(a.direction-(90*a.cortisol), 360))
		} else {
			a.move(mod(a.direction+(90*a.cortisol), 360))
		}
	}
}

func (a *Agent) findEatFood(agents, foods, walls []web_lib.Agent) {
	if foods == nil {
		// dont see food
		// if see agents with AgentVal < 0 turn away
		flag := false
		flag2 := false
		for _, temp := range agents {
			tmpAgentVal := a.agentVal(temp.(*Agent))
			if tmpAgentVal < 0 && a.stressed {
				if randBool() {
					a.move(mod(a.direction-(90*1.5*a.cortisol), 360))
				} else {
					a.move(mod(a.direction+(90*1.5*a.cortisol), 360))
				}
				flag = true
				break
			} else if tmpAgentVal < 0 {
				flag2 = true
			}
		}

		if !flag {
			// if not stressed and no high ranked agents but see wall turn away else random move
			if !flag2 {
				if walls != nil {
					a.turnFromWall()
				} else {
					a.randomMove()
				}
			} else {
				// higher ranked agents but not stressed
				if randBool() {
					a.move(mod(a.direction-(90*a.cortisol), 360))
				} else {
					a.move(mod(a.direction+(90*a.cortisol), 360))
				}
			}
		}
	} else {
		// see food so approach or eat if close
		// first calculate closest food
		var food web_lib.Agent
		dist := 100.0
		for _, temp := range foods {
			tmpDist := distance(a.x, a.y, temp.X(), temp.Y())
			if tmpDist < dist {
				food = temp
				dist = tmpDist
			}
		}
		if dist <= 1 {
			// next to food, so can eat
			a.eatFood(food.(*Food))
			a.justEaten = true
			a.checkEatenWithBondPartner(food.(*Food))
			if a.energy >= 1 {
				if randBool() {
					a.move(mod(a.direction-90, 360))
				} else {
					a.move(mod(a.direction+90, 360))
				}
				a.energy = 1
				a.justEaten = false
				a.sharedEatingFood()
				a.sharedFoodWith = nil
				a.foodTimeWaiting = 0
			}
		} else {
			// see food, calculate if can approach
			a.approachOrAvoid(food.(*Food))
		}
	}
}

func (a *Agent) eatFood(f *Food) {
	if a.foodTimeWaiting < 5 {
		a.foodTimeWaiting++
	} else {
		f.reduceResource(0.01)
		a.energy += 0.01
		a.foodTimeWaiting++
	}
	if a.foodTimeWaiting >= 6 {
		a.foodTimeWaiting = 0
	}
}

func (a *Agent) turnFromWall() {
	a.direction = mod(a.direction+rand.Float64()*135-rand.Float64()*135, 360)
}

func (a *Agent) approachOrAvoid(f *Food) {
	agentVal := 1.0
	if f.Owner() != nil {
		if f.Owner() != a {
			agentVal = a.agentVal(f.Owner())
		}
	}
	if agentVal < 0 {
		// cant approach food
		a.move(mod(a.direction-180, 360))
	} else {
		a.moveTo(f)
	}
}

func (a *Agent) moveTo(agent web_lib.Agent) {
	a.move(math.Atan2(agent.X()-a.x, agent.Y()-a.y) * (180.0 / math.Pi))
}

func (a *Agent) randomMove() {
	a.move(mod(a.direction+rand.Float64()*20-rand.Float64()*20, 360))
}

func (a *Agent) move(direction float64) {
	oldx, oldy := a.x, a.y
	oldDirection := a.direction
	a.direction = direction
	if a.stressed {
		a.stepSize = 0.5 + a.cortisol*1.25
	} else {
		a.stepSize = 0.5 + a.cortisol*0.75
	}
	a.x = oldx + a.stepSize*math.Sin(a.direction*(math.Pi/180.0))
	a.y = oldy + a.stepSize*math.Cos(a.direction*(math.Pi/180.0))

	var err error
	if a.trail {
		err = a.grid.Copy(a.id, oldx, oldy, a.x, a.y)
	} else {
		err = a.grid.Move(a.id, oldx, oldy, a.x, a.y)
	}

	if err != nil {
		a.x, a.y = oldx, oldy
		a.direction = oldDirection
	}
}

func (a *Agent) checkEatenWithBondPartner(food *Food) {
	for _, agent := range food.EatingAgents() {
		for _, id := range a.bondPartners {
			if agent.ID() == id {
				// there is a bond with some other eating agent
				if !inList(agent.ID(), a.sharedFoodWith) {
					a.sharedFoodWith = append(a.sharedFoodWith, agent.ID())
				}
			}
		}
	}
}

func (a *Agent) sharedEatingFood() {
	if a.sharedFoodWith == nil {
		return
	}
	oxyGain := 2 - (2*a.oxytocin)*0.2
	a.IncreaseOT(oxyGain)
	if a.DSImode == "Variable" {
		for _, id := range a.sharedFoodWith {
			a.ModulateDSI(id, a.tactileEat*0.3)
		}
	}
}

func (a *Agent) updateInternals() {
	a.mutex.Lock()
	// lose energy
	a.energy = a.energy - (a.nutritionChange * a.stepSize)
	// lose and correct socialness
	if a.socialness > 1 {
		a.socialness = 1
	}
	a.socialness -= a.socialChange
	if a.socialness < 0 {
		a.socialness = 0
	}
	// lose and correct oxytocin
	if a.oxytocin > 1 {
		a.oxytocin = 1
	}
	a.oxytocin -= a.oxytocinChange
	if a.oxytocin < 0 {
		a.oxytocin = 0
	}
	// correct DSIstrengts
	for i := 0; i < len(a.DSIstrengths); i++ {
		if a.DSIstrengths[i] > 2 {
			a.DSIstrengths[i] = 2
		}
		if a.DSImode == "Variable" {
			a.DSIstrengths[i] = a.DSIstrengths[i] * 0.9997
		}
		if a.DSIstrengths[i] < 0 {
			a.DSIstrengths[i] = 0
		}
	}
	// checked if stressed
	if a.cortisol > a.adaptiveThreshold {
		a.stressed = true
	} else {
		a.stressed = false
	}
	a.mutex.Unlock()
}

func (a *Agent) updateCT(sumOfErrors float64, agents, foods []web_lib.Agent) {
	availableAgents := 0.0
	availableFoods := 0.0
	finalAgentVal := 0.0
	if len(agents) > 0 {
		agentVal := -1.0
		rankDiff := 0.0
		bond := 0.0
		DSI := 0.0
		for _, temp := range agents {
			for i, id := range a.bondPartners {
				if temp.ID() == id {
					// there is a bond
					bond = 1.0
					tmpDSI := a.DSIstrengths[i]
					if tmpDSI > DSI {
						DSI = tmpDSI
					}
					break
				}
			}
			tmpRankDiff := float64(temp.(*Agent).Rank()-a.rank) / float64(a.numOfAgents-1)
			tmpAgentVal := a.agentVal(temp.(*Agent))
			if tmpAgentVal >= agentVal {
				agentVal = tmpAgentVal
			}
			if tmpRankDiff >= rankDiff {
				rankDiff = tmpRankDiff
			}
		}
		availableAgents = (1 - rankDiff) + bond*DSI*a.oxytocin
		finalAgentVal = agentVal
	}

	if len(foods) > 0 && finalAgentVal >= 0 {
		availableFoods = 1.0
	}

	releaseRateCT := ((sumOfErrors - availableAgents - availableFoods) / 2) * a.cortisolChange
	if releaseRateCT < 0 {
		releaseRateCT = releaseRateCT / 2
	}
	a.mutex.Lock()
	a.cortisol = a.cortisol + releaseRateCT
	if a.cortisol < 0 {
		a.cortisol = 0
	}
	if a.cortisol > 1 {
		a.cortisol = 1
	}
	if a.cortisol > a.adaptiveThreshold {
		a.stressed = true
	} else {
		a.stressed = false
	}
	a.mutex.Unlock()
}

func (a *Agent) inVision() ([]web_lib.Agent, []web_lib.Agent, []web_lib.Agent) {
	var foods, agents, walls []web_lib.Agent
	for _, agent := range a.grid.agentVision[a.id-1] {
		if agent.ID() == -1 {
			// sees food
			foods = append(foods, agent)
		} else if agent.ID() == -3 {
			// sees a wall
			walls = append(walls, agent)
		} else {
			// the agent sees another agent
			agents = append(agents, agent)
		}
	}
	return foods, agents, walls
}

/*
   ________________________________________________________________________________________________________________________
   ___________________________________________SETUP/GETTERS/SETTERS________________________________________________________
   ________________________________________________________________________________________________________________________
*/

func inList(element int, list []int) bool {

	for _, temp := range list {
		if temp == element {
			return true
		}
	}
	return false
}

func mod(a, b float64) float64 {
	c := math.Mod(a, b)
	if c < 0 {
		return c + b
	}
	return c
}

func distance(x1, y1, x2, y2 float64) float64 {
	return math.Sqrt(math.Pow(x2-x1, 2) + math.Pow(y2-y1, 2))
}

func cortisolThreshold(rank int, CortisolThresholdCondition string) (float64, error) {
	switch CortisolThresholdCondition {
	case "Control":
		return 1.1, nil
	case "Neutral":
		return 0.5, nil
	case "High":
		return 0.7, nil
	case "Low":
		return 0.2, nil
	case "Low-High":
		switch rank {
		case 1:
			return 0.7, nil
		case 2:
			return 0.6, nil
		case 3:
			return 0.5, nil
		case 4:
			return 0.4, nil
		case 5:
			return 0.3, nil
		case 6:
			return 0.2, nil
		}
	case "High-Low":
		switch rank {
		case 1:
			return 0.2, nil
		case 2:
			return 0.3, nil
		case 3:
			return 0.4, nil
		case 4:
			return 0.5, nil
		case 5:
			return 0.6, nil
		case 6:
			return 0.7, nil
		}
	}
	return 0, errors.New("invalid cortisol threshhold condition")
}

func (a *Agent) SetBonds(bonds []int) {
	a.bondPartners = bonds
	for i := 0; i < len(bonds); i++ {
		a.DSIstrengths = append(a.DSIstrengths, 2)
	}
}

func (a *Agent) ModulateDSI(id int, amount float64) {
	a.mutex.Lock()
	for i, partnerID := range a.bondPartners {
		if id == partnerID {
			a.DSIstrengths[i] = a.DSIstrengths[i] + amount
			if a.DSIstrengths[i] > 2 {
				a.DSIstrengths[i] = 2
			} else if a.DSIstrengths[i] < 0 {
				a.DSIstrengths[i] = 0
			}
			break
		}
	}
	a.mutex.Unlock()
}

func (a *Agent) ModulateCT(amount float64) {
	a.mutex.Lock()
	a.cortisol = a.cortisol + amount
	if a.cortisol < 0 {
		a.cortisol = 0
	}
	if a.cortisol > 1 {
		a.cortisol = 1
	}
	a.mutex.Unlock()
}
func (a *Agent) IncreaseOT(intensity float64) {
	a.mutex.Lock()
	a.oxytocin = a.oxytocin + intensity
	if a.oxytocin > 1 {
		a.oxytocin = 1
	}
	a.mutex.Unlock()
}

func randBool() bool {
	return rand.Float32() < 0.5
}

func (a *Agent) Rank() int          { return a.rank }
func (a *Agent) ID() int            { return a.id }
func (a *Agent) Direction() float64 { return a.direction }
func (a *Agent) Alive() bool        { return a.alive }
func (a *Agent) X() float64         { return a.x }
func (a *Agent) Y() float64         { return a.y }
