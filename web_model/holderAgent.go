package web_model

import (
	"github.com/Kubiuks/Alife_web/web_lib"
	"sync"
)

type HolderAgent struct {
	mx     sync.RWMutex
	id int
	x,y float64
	grid *Grid
	agents []web_lib.Agent
}

func NewHolderAgent(grid *Grid, x, y float64) *HolderAgent {
	return &HolderAgent{
		id:    -2,
		x:     x,
		y:     y,
		grid:  grid,
	}
}

func (h *HolderAgent) AddAgent(agent web_lib.Agent) {
	h.mx.Lock()
	h.agents = append(h.agents, agent)
	h.mx.Unlock()
}

func (h *HolderAgent) DeleteAgent(id int) (web_lib.Agent, web_lib.Agent){
	h.mx.Lock()
	defer h.mx.Unlock()
	l := len(h.agents)
	for i, e := range h.agents{
		if e.ID() == id {
			h.agents[i] = h.agents[l-1]
			if l == 2{
				return h.agents[0], e
			} else {
				h.agents = h.agents[:l-1]
				return h, e
			}
		}
	}
	return h, nil
}

func (h *HolderAgent) Alive() bool { return false }
func (h *HolderAgent) ID() int { return h.id }
func (h *HolderAgent) X() float64 { return h.x }
func (h *HolderAgent) Y() float64 { return h.y }

func (h *HolderAgent) Agents() []web_lib.Agent {
	h.mx.RLock()
	defer h.mx.RUnlock()
	return h.agents
}

func (h *HolderAgent) Run(){
	return
}