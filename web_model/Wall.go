package web_model

type Wall struct {
	id 			 int
	x, y         float64
}

func NewWall(x, y float64) *Wall {
	return &Wall{
		id:    -3,
		x:     x,
		y:     y,
	}
}

func (w *Wall) Alive() bool { return false }
func (w *Wall) Run() { return }
func (w *Wall) ID() int { return w.id }
func (w *Wall) X() float64 { return w.x }
func (w *Wall) Y() float64 { return w.y }