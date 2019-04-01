package common

import (
	"net/http"

	"github.com/go-redis/redis"
	mgo "gopkg.in/mgo.v2"

	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/ws"
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
	UpNext      *models.Run
	RedisClient *redis.Client
	TimerState  TimerState
	TimerTime   float64
	HTTPClient  http.Client
	// SocialUpdatesChan is used to communicate with the socialController on twitter and twitch updates
	SocialUpdatesChan chan int
	CL                *Checklist
	Settings          *SettingsProvider
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
	// TimerRunning represents a running timer
	TimerRunning TimerState = iota
	// TimerPaused represents a paused timer
	TimerPaused
	// TimerStopped represents a stopped timer
	TimerStopped
	// TimerFinished represents a finished timer
	TimerFinished
)

// NewController returns a new base controller
func NewController(hub *ws.Hub, mgs *mgo.Session, crIndex int, rc *redis.Client) *Controller {
	runs := []models.Run{}
	mgs.DB("marathon").C("runs").Find(nil).All(&runs)
	c := &Controller{
		WS:                hub,
		MGS:               mgs,
		RunIndex:          crIndex,
		Col:               mgs.DB("marathon").C("runs"),
		RedisClient:       rc,
		TimerState:        2,
		TimerTime:         0,
		HTTPClient:        http.Client{},
		SocialUpdatesChan: make(chan int, 1),
	}
	c.CL = NewChecklist(c)
	c.Settings = InitSettings(c)
	c.UpdateActiveRuns()
	c.UpdateUpNext()
	return c
}
