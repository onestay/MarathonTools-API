package common

import (
	"encoding/json"
	"net/http"

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
}

type httpResponse struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

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
	}

	c.UpdateActiveRuns()

	return c
}

// UpdateActiveRuns will update the the previous, current and next run in the base controller struct
func (c *Controller) UpdateActiveRuns() {
	runs := []models.Run{}
	c.Col.Find(nil).All(&runs)
	c.CurrentRun = &runs[c.RunIndex]

	if c.RunIndex == 0 {
		c.PrevRun = &models.Run{}
	} else {
		c.PrevRun = &runs[c.RunIndex-1]
	}
	if len(runs) <= c.RunIndex+1 {
		c.NextRun = &models.Run{}
	} else {
		c.NextRun = &runs[c.RunIndex+1]
	}
}

// Response will send out a generic response
func (c Controller) Response(res, err string, code int, w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	ok := true
	if err != "" {
		ok = false
	}

	resStruct := httpResponse{
		Ok:   ok,
		Data: res,
		Err:  err,
	}

	json.NewEncoder(w).Encode(resStruct)
}

// SendInitialData will send some initial data over the websocket
func (c Controller) SendInitialData() []byte {
	runs := []models.Run{}
	c.Col.Find(nil).All(&runs)

	data := struct {
		DataType   string       `json:"dataType"`
		Runs       []models.Run `json:"runs"`
		PrevRun    models.Run   `json:"prevRun"`
		CurrentRun models.Run   `json:"currentRun"`
		NextRun    models.Run   `json:"nextRun"`
		RunIndex   int          `json:"runIndex"`
	}{"initalData", runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex}

	d, _ := json.Marshal(data)

	return d
}

// WSRunUpdate sends an update for all runs over the websocket and current runs over the websocket.
func (c Controller) WSRunUpdate() {
	runs := []models.Run{}
	c.Col.Find(nil).All(&runs)

	data := struct {
		DataType   string       `json:"dataType"`
		Runs       []models.Run `json:"runs"`
		PrevRun    models.Run   `json:"prevRun"`
		CurrentRun models.Run   `json:"currentRun"`
		NextRun    models.Run   `json:"nextRun"`
		RunIndex   int          `json:"runIndex"`
	}{"runUpdate", runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSRunsOnlyUpdate only updates runs and not current runs
func (c Controller) WSRunsOnlyUpdate() {
	runs := []models.Run{}
	c.Col.Find(nil).All(&runs)

	data := struct {
		DataType string       `json:"dataType"`
		Runs     []models.Run `json:"runs"`
	}{"runUpdate", runs}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSCurrentUpdate sends ws data with the current runs
func (c Controller) WSCurrentUpdate() {
	data := struct {
		DataType   string     `json:"dataType"`
		PrevRun    models.Run `json:"prevRun"`
		CurrentRun models.Run `json:"currentRun"`
		NextRun    models.Run `json:"nextRun"`
		RunIndex   int        `json:"runIndex"`
	}{"runUpdate", *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSTimeUpdate sends a time update
// We do that here because timerState is not defined in the base controller and I'm zoo lazy to change everything
func (c Controller) WSTimeUpdate(time float64) {
	data := struct {
		DataType string  `json:"dataType"`
		T        float64 `json:"t"`
	}{"timeUpdate", time}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}
