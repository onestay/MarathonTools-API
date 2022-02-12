package common

import (
	"database/sql"
	"github.com/onestay/MarathonTools-API/marathon"
	"net/http"

	"github.com/go-redis/redis"
	"gopkg.in/mgo.v2"

	"github.com/onestay/MarathonTools-API/ws"
)

// Controller is the base struct for any controller. It's used to manage state and other things.
type Controller struct {
	WS          *ws.Hub
	RedisClient *redis.Client
	TimerState  TimerState
	TimerTime   float64
	HTTPClient  http.Client
	Marathon    marathon.Marathon
	// SocialUpdatesChan is used to communicate with the socialController on Twitter and twitch updates
	SocialUpdatesChan chan int
	CL                *Checklist
	Settings          *SettingsProvider
	db                *sql.DB
}

type httpResponse struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

// The reason why all the timer stuff is here is to make it accessible to all other controllers which
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
func NewController(hub *ws.Hub, mgs *mgo.Session, crIndex int, rc *redis.Client) (*Controller, error) {
	db, err := sql.Open("sqlite3", "../../db/test.sqlite")
	if err != nil {
		return nil, err
	}

	c := &Controller{
		WS:                hub,
		RedisClient:       rc,
		TimerState:        2,
		TimerTime:         0,
		HTTPClient:        http.Client{},
		SocialUpdatesChan: make(chan int, 1),
		db:                db,
	}
	c.CL = NewChecklist(c)
	c.Settings = InitSettings(c)
	c.UpdateActiveRuns()
	c.UpdateUpNext()
	return c, nil
}
