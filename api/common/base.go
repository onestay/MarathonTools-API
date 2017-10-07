package common

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/ws"
	"gopkg.in/mgo.v2"
)

// Controller is the base struct for any controller. It's used to manage state and other things.
type Controller struct {
	WS         *ws.Hub
	MGS        *mgo.Session
	Col        *mgo.Collection
	RunIndex   int
	CurrentRun *models.Run
	NextRun    *models.Run
	PrevRun    *models.Run
}

type httpResponse struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

// NewController returns a new base controlle
func NewController(hub *ws.Hub, mgs *mgo.Session, crIndex int) *Controller {
	runs := []models.Run{}
	mgs.DB("marathon").C("runs").Find(nil).All(&runs)
	c := &Controller{
		WS:       hub,
		MGS:      mgs,
		RunIndex: crIndex,
		Col:      mgs.DB("marathon").C("runs"),
	}

	c.UpdateActiveRuns()

	return c
}

// UpdateActiveRuns will update the the previous, current and next run in the base controller struct
func (c *Controller) UpdateActiveRuns() {
	runs := []models.Run{}
	c.Col.Find(nil).All(&runs)
	fmt.Println(runs[c.RunIndex])
	c.CurrentRun = &runs[0]

	if c.RunIndex == 0 {
		c.PrevRun = &models.Run{}
	} else {
		c.PrevRun = &runs[c.RunIndex-1]
	}
	if len(runs) < c.RunIndex+1 {
		c.NextRun = &models.Run{}
	} else {
		c.NextRun = &runs[c.RunIndex+1]
	}
}

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
