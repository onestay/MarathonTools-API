package common

import (
	"encoding/json"
	"net/http"

	"github.com/onestay/MarathonTools-API/api/models"
)

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
