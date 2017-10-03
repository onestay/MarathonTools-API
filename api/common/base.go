package common

import (
	"encoding/json"
	"net/http"

	"github.com/onestay/MarathonTools-API/ws"
	"gopkg.in/mgo.v2"
)

// Controller is the base struct for any controller. It's used to manage state and other things.
type Controller struct {
	WS  *ws.Hub
	MGS *mgo.Session
}

type httpResponse struct {
	Ok   bool   `json:"ok"`
	Data string `json:"data,omitempty"`
	Err  string `json:"error,omitempty"`
}

// NewController returns a new base controlle
func NewController(hub *ws.Hub, mgs *mgo.Session) *Controller {
	return &Controller{
		WS:  hub,
		MGS: mgs,
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
