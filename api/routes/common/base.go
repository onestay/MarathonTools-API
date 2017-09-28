package common

import (
	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/ws"
	"gopkg.in/mgo.v2"
)

// Controller is the base struct for any controller. It's used to manage state and other things.
type Controller struct {
	WS       *ws.Hub
	Marathon *models.Marathon
	MGS      *mgo.Session
}

// NewController returns a new base controlle
func NewController(hub *ws.Hub, marathon *models.Marathon, mgs *mgo.Session) *Controller {
	return &Controller{
		WS:       hub,
		Marathon: marathon,
		MGS:      mgs,
	}
}
