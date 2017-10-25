package timer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
)

type timerState int

const (
	running timerState = iota
	paused
	stopped
	finished
)

// Controller is the time controller
type Controller struct {
	b               *common.Controller
	ticker          *time.Ticker
	refreshInterval int
	startTime       time.Time
	lastPaused      time.Time
	state           timerState
	time            float64
}

// NewTimeController initializes and returns a new time controller. The refreshInterval is in ms
func NewTimeController(b *common.Controller, refreshInterval int) *Controller {
	return &Controller{
		b:               b,
		state:           stopped,
		refreshInterval: refreshInterval,
		time:            0,
	}
}

func (c *Controller) timerLoop() {
	c.ticker = time.NewTicker(100 * time.Millisecond)

	go func() {
		for {
			select {
			case t := <-c.ticker.C:
				difference := t.Sub(c.startTime).Seconds()
				go c.b.WSTimeUpdate(difference)
				c.time = difference
			}
		}
	}()
}

// TimerStart will start the timer
// req state: stopped
func (c *Controller) TimerStart(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("start", w) {
		return
	}

	c.startTime = time.Now()
	c.timerLoop()

	c.state = running
	c.wsStateUpdate()

	w.WriteHeader(http.StatusNoContent)
}

// TimerPause will pause the timer
// req state: running
func (c *Controller) TimerPause(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("pause", w) {
		return
	}

	c.lastPaused = time.Now()
	c.ticker.Stop()

	c.state = paused
	c.wsStateUpdate()

	w.WriteHeader(http.StatusNoContent)
}

// TimerResume will resume the timer
// req state: finished, pause
func (c *Controller) TimerResume(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("resume", w) {
		return
	}

	c.startTime = c.startTime.Add(time.Since(c.lastPaused))
	c.timerLoop()

	c.state = running
	c.wsStateUpdate()

	w.WriteHeader(http.StatusNoContent)
}

// TimerFinish will be fired when all players are done, can also be manually called
// req state: running
func (c *Controller) TimerFinish(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("finish", w) {
		return
	}

	c.ticker.Stop()

	c.state = finished
	c.wsStateUpdate()

	w.WriteHeader(http.StatusNoContent)
}

// TimerReset will reset the timer
// req state: finished
func (c *Controller) TimerReset(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("reset", w) {
		return
	}
	c.ticker.Stop()

	for i := 0; i < len(c.b.CurrentRun.Players); i++ {
		c.b.CurrentRun.Players[i].Timer.Finished = false
		c.b.CurrentRun.Players[i].Timer.Time = 0
	}

	c.state = stopped
	c.wsStateUpdate()

	w.WriteHeader(http.StatusNoContent)
	c.b.WSCurrentUpdate()

}

// TimerPlayerFinish will finish a specific player
// req state: running
func (c *Controller) TimerPlayerFinish(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	if c.invalidState("playerFinish", w) {
		return
	}

	pID, err := strconv.Atoi(ps.ByName("id"))
	if err != nil {
		c.b.Response("", "id not provided or not valid int", 400, w)
		return
	}
	c.b.CurrentRun.Players[pID].Timer.Finished = true
	c.b.CurrentRun.Players[pID].Timer.Time = c.time
	c.b.WSCurrentUpdate()

	for i := 0; i < len(c.b.CurrentRun.Players); i++ {
		if c.b.CurrentRun.Players[i].Timer.Finished != true {
			break
		}
		c.TimerFinish(w, r, ps)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (c *Controller) invalidState(method string, w http.ResponseWriter) bool {
	fmt.Println(c.state)
	f := true
	if method == "start" && c.state == stopped {
		f = false
	} else if method == "finish" && c.state == running {
		f = false
	} else if method == "resume" && c.state == paused || c.state == finished {
		f = false
	} else if method == "playerFinish" && c.state == running {
		f = false
	} else if method == "pause" && c.state == running {
		f = false
	} else if method == "reset" && c.state == finished {
		f = false
	}

	if f {
		c.b.Response("", fmt.Sprintf("method %v not allowed with state %v", method, c.state), 400, w)
	}
	return f
}

func (c Controller) wsStateUpdate() {
	data := struct {
		DataType string     `json:"dataType"`
		State    timerState `json:"state"`
	}{"stateUpdate", c.state}

	d, _ := json.Marshal(data)

	c.b.WS.Broadcast <- d
}
