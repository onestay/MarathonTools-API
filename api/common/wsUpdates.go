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
	}{"initalData", runs, *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex, c.TimerState}

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
		RunIndex   int        `json:"runIndex"`
	}{"runCurrentUpdate", *c.PrevRun, *c.CurrentRun, *c.NextRun, c.RunIndex}

	d, _ := json.Marshal(data)

	c.WS.Broadcast <- d
}

// WSTimeUpdate sends a time update
func (c Controller) WSTimeUpdate(time float64) {
	data := struct {
		DataType string  `json:"dataType"`
		T        float64 `json:"t"`
	}{"timeUpdate", time}

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
