package common

import (
	"encoding/json"
	"fmt"
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
	fmt.Println(c.RunIndex)
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
		Runs       []models.Run `json:"runs,omitempty"`
		PrevRun    models.Run   `json:"prevRun,omitempty"`
		CurrentRun models.Run   `json:"currentRun,omitempty"`
		NextRun    models.Run   `json:"nextRun,omitempty"`
		RunIndex   int          `json:"runIndex,omitempty"`
	}{runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex}

	d, _ := json.Marshal(data)

	return d
}
