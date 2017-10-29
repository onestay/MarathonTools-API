package common

import (
	"github.com/go-redis/redis"

	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/ws"

	"gopkg.in/mgo.v2"
)

// Controller is the base struct for any controller. It's used to manage state and other things.
type Controller struct {
	WS          *ws.Hub
	MGS         *mgo.Session
	Col         *mgo.Collection
	RunIndex    int
	CurrentRun  *models.Run
	NextRun     *models.Run
	PrevRun     *models.Run
	RedisClient *redis.Client
	TimerState  TimerState
	TimerTime   float64
}

type httpResponse struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

// The reason why all the timer stuff is here is to make it accesible to all other controllers which
// may need access to timer states and constants
// Especially the base controller, because it couldn't import the timerState from the timer class

// TimerState is an alias of int to represent timer state
type TimerState = int

const (
	// Running represents a running timer
	Running TimerState = iota
	// Paused represents a paused timer
	Paused
	// Stopped represents a stopped timer
	Stopped
	// Finished represents a finished timer
	Finished
)

// NewController returns a new base controlle
func NewController(hub *ws.Hub, mgs *mgo.Session, crIndex int, rc *redis.Client) *Controller {
	runs := []models.Run{}
	mgs.DB("marathon").C("runs").Find(nil).All(&runs)
	c := &Controller{
		WS:          hub,
		MGS:         mgs,
		RunIndex:    crIndex,
		Col:         mgs.DB("marathon").C("runs"),
		RedisClient: rc,
		TimerState:  0,
		TimerTime:   0,
	}

	c.UpdateActiveRuns()

	return c
}
