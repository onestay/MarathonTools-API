package common

import (
	"encoding/json"

	"github.com/onestay/MarathonTools-API/api/models"
)

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
		TimerState TimerState   `json:"timerState"`
		UpNextRun  models.Run   `json:"upNext"`
	}{"initalData", runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex, c.TimerState, *c.UpNext}

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
		UpNextRun  models.Run   `json:"upNext"`
	}{"runUpdate", runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex, *c.UpNext}

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
	}{"runsOnlyUpdate", runs}

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
		UpNextRun  models.Run `json:"upNext"`
		RunIndex   int        `json:"runIndex"`
	}{"runCurrentUpdate", *c.PrevRun, *c.CurrentRun, *c.NextRun, *c.UpNext, c.RunIndex}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSTimeUpdate sends a time update
func (c Controller) WSTimeUpdate() {
	data := struct {
		DataType string  `json:"dataType"`
		T        float64 `json:"t"`
	}{"timeUpdate", c.TimerTime}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSStateUpdate sends a state update
func (c Controller) WSStateUpdate() {
	data := struct {
		DataType string     `json:"dataType"`
		State    TimerState `json:"state"`
	}{"stateUpdate", c.TimerState}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSReportError provides a helper to send an error message to the client
func (c Controller) WSReportError(e string) {
	data := struct {
		DataType string `json:"dataType"`
		Error    string `json:"error"`
	}{"error", e}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

func (c Controller) WSDonationUpdate(oldAmount, newAmount float64) {
	data := struct {
		DataType   string  `json:"dataType"`
		OldAmount  float64 `json:"old"`
		NewAmount  float64 `json:"new"`
		Difference float64 `json:"difference"`
	}{"donationUpdate", oldAmount, newAmount, newAmount - oldAmount}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}
