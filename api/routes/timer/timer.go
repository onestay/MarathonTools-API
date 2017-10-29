package timer

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
)

// Controller is the time controller
type Controller struct {
	b               *common.Controller
	ticker          *time.Ticker
	refreshInterval int
	startTime       time.Time
	lastPaused      time.Time
}

// NewTimeController initializes and returns a new time controller. The refreshInterval is in ms
func NewTimeController(b *common.Controller, refreshInterval int) *Controller {
	return &Controller{
		b:               b,
		refreshInterval: refreshInterval,
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
				c.b.TimerTime = difference
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

	c.b.TimerState = common.Running
	c.b.WSStateUpdate()

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

	c.b.TimerState = common.Paused
	c.b.WSStateUpdate()

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

	c.b.TimerState = common.Running
	c.b.WSStateUpdate()

	w.WriteHeader(http.StatusNoContent)
}

// TimerFinish will be fired when all players are done, can also be manually called
// req state: running
func (c *Controller) TimerFinish(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if c.invalidState("finish", w) {
		return
	}

	c.ticker.Stop()

	c.b.TimerState = common.Finished
	c.b.WSStateUpdate()

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

	c.b.TimerState = common.Stopped
	c.b.WSStateUpdate()

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
	c.b.CurrentRun.Players[pID].Timer.Time = c.b.TimerTime
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
	fmt.Println(c.b.TimerState)
	f := true
	if method == "start" && c.b.TimerState == common.Stopped {
		f = false
	} else if method == "finish" && c.b.TimerState == common.Running {
		f = false
	} else if method == "resume" && c.b.TimerState == common.Paused || c.b.TimerState == common.Finished {
		f = false
	} else if method == "playerFinish" && c.b.TimerState == common.Running {
		f = false
	} else if method == "pause" && c.b.TimerState == common.Running {
		f = false
	} else if method == "reset" && c.b.TimerState == common.Finished {
		f = false
	}

	if f {
		c.b.Response("", fmt.Sprintf("method %v not allowed with state %v", method, c.b.TimerState), 400, w)
	}
	return f
}
