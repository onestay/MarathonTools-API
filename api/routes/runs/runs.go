package runs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/api/routes/social"
	"gopkg.in/mgo.v2/bson"
)

// RunController contains all the methods needed to control runs
type RunController struct {
	base *common.Controller
}

func (rc RunController) registerRoutes(r *httprouter.Router) {
	r.GET("/run/get/all", rc.GetRuns)
	r.GET("/run/get/single/:id", rc.GetRun)
	r.GET("/run/get/active", rc.ActiveRuns)

	r.DELETE("/run/delete/:id", rc.DeleteRun)

	r.PATCH("/run/update/:id", rc.UpdateRun)

	r.POST("/run/move/:id/:after", rc.MoveRun)
	r.POST("/run/add/single", rc.AddRun)
	r.POST("/run/switch", rc.SwitchRun)

	r.POST("/run/layout", rc.RefreshLayout)
	r.POST("/run/upload", rc.UploadRunJSON)

}

// NewRunController returns a new run controller
func NewRunController(b *common.Controller, router *httprouter.Router) {
	r := RunController{
		base: b,
	}

	r.registerRoutes(router)
}

// RefreshLayout will send a WsCurrentUpdate to refresh the layout
func (rc RunController) RefreshLayout(_ http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	rc.base.WSCurrentUpdate()
}

// AddRun will add a run to the database and return the ID of the new run
func (rc RunController) AddRun(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	run := models.Run{}
	json.NewDecoder(r.Body).Decode(&run)

	run.RunID = bson.NewObjectId()

	err := rc.base.MGS.DB("marathon").C("runs").Insert(run)
	if err != nil {
		rc.base.Response("", "err adding run", http.StatusInternalServerError, w)
		return
	}

	rc.base.Response(run.RunID.Hex(), "", http.StatusOK, w)

	go rc.base.WSRunsOnlyUpdate()
	go rc.base.UpdateActiveRuns()
}

// GetRuns will return all runs from the mgo collection
func (rc RunController) GetRuns(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	var runs []models.Run

	err := rc.base.MGS.DB("marathon").C("runs").Find(nil).All(&runs)
	if err != nil {
		rc.base.Response("", err.Error(), http.StatusInternalServerError, w)
		fmt.Println(err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(runs)
}

// GetRun will return a run
func (rc RunController) GetRun(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
	runID := ps.ByName("id")
	if !bson.IsObjectIdHex(runID) {
		rc.base.Response("", "invalid bson id", http.StatusBadRequest, w)
		return
	}

	run := models.Run{}
	err := rc.base.MGS.DB("marathon").C("runs").FindId(bson.ObjectIdHex(runID)).One(&run)

	if s := err.Error(); s == "not found" {
		rc.base.Response("", err.Error(), http.StatusNotFound, w)
		return
	} else if err != nil {
		rc.base.Response("", err.Error(), http.StatusInternalServerError, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(run)
}

func (rc RunController) ActiveRuns(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	runs := make([]models.Run, 3)

	runs[0] = *rc.base.PrevRun
	runs[1] = *rc.base.CurrentRun
	runs[2] = *rc.base.NextRun

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	json.NewEncoder(w).Encode(&runs)
}

// DeleteRun will delete a run with the provided id
func (rc RunController) DeleteRun(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
	runID := ps.ByName("id")
	if !bson.IsObjectIdHex(runID) {
		rc.base.Response("", "invalid bson id", http.StatusBadRequest, w)
		return
	}

	err := rc.base.MGS.DB("marathon").C("runs").RemoveId(bson.ObjectIdHex(runID))
	if err != nil {
		fmt.Println(err)
		rc.base.Response("", err.Error(), http.StatusNotFound, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	rc.base.WSRunUpdate()
}

// UpdateRun will update the run with the id provided and the request body
func (rc RunController) UpdateRun(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	runID := ps.ByName("id")
	if !bson.IsObjectIdHex(runID) {
		rc.base.Response("", "invalid bson id", http.StatusBadRequest, w)
		return
	}

	updatedRun := models.Run{}

	err := json.NewDecoder(r.Body).Decode(&updatedRun)
	if err != nil {
		rc.base.Response("", "couldn't unmarshal body", http.StatusInternalServerError, w)
		log.Printf("Error in UpdateRun: %v", err)
		return
	}
	updatedRun.RunID = bson.ObjectIdHex(runID)

	err = rc.base.MGS.DB("marathon").C("runs").UpdateId(bson.ObjectIdHex(runID), updatedRun)
	if err != nil {
		rc.base.Response("", err.Error(), http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)

	rc.base.WSRunsOnlyUpdate()
	if updatedRun.RunID == rc.base.CurrentRun.RunID {
		rc.base.CurrentRun = &updatedRun
		rc.base.WSCurrentUpdate()
	}

}

// MoveRun takes the run by id and moves it after the run provided by after
// to do this we have to pull every run from the collection, then delete every run in the db
// do the moving and insert all the records into the db again
// which is kinda stupid tbh
func (rc RunController) MoveRun(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
	runID := ps.ByName("id")
	after := ps.ByName("after")
	if !bson.IsObjectIdHex(runID) || !bson.IsObjectIdHex(after) {
		rc.base.Response("", "invalid bson id", http.StatusBadRequest, w)
		return
	}

	var runs []models.Run

	err := rc.base.MGS.DB("marathon").C("runs").Find(nil).All(&runs)
	if err != nil {
		rc.base.Response("", err.Error(), http.StatusInternalServerError, w)
		fmt.Println(err)
		return
	}

	var index int
	var indexToInsert int

	for i := 0; i < len(runs); i++ {
		if runs[i].RunID == bson.ObjectIdHex(runID) {
			index = i
		} else if runs[i].RunID == bson.ObjectIdHex(after) {
			indexToInsert = i
		}
	}

	q := runs[index]
	b := append(runs[:index], runs[index+1:]...)
	runs = append(b[:indexToInsert], append([]models.Run{q}, b[indexToInsert:]...)...)

	rc.base.MGS.DB("marathon").C("runs").RemoveAll(nil)

	for i := 0; i < len(runs); i++ {
		rc.base.MGS.DB("marathon").C("runs").Insert(runs[i])
	}

	w.WriteHeader(http.StatusNoContent)

	rc.base.WSRunsOnlyUpdate()
}

// SwitchRun will update the currently active, upcoming and previous run based on the current run index
func (rc *RunController) SwitchRun(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if rc.base.TimerState != common.TimerStopped {
		rc.base.Response("", "can't switch runs while timer is running", 400, w)
		return
	}

	meth := "next"
	if r.URL.Query().Get("m") == "prev" {
		meth = "prev"
	}
	c, _ := rc.base.Col.Count()

	if meth == "next" {
		if c <= rc.base.RunIndex+1 {
			rc.base.Response("", "no next run", 400, w)
			return
		}
		rc.base.RunIndex++
	} else if meth == "prev" {
		if rc.base.RunIndex == 0 {
			rc.base.Response("", "no prev run", 400, w)
			return
		}
		rc.base.RunIndex--
	}

	rc.base.UpdateActiveRuns()
	if rc.base.CL.CheckDone() {
		go rc.checkForUpdate()
	}
	go rc.base.CL.ResetChecklist()

	w.WriteHeader(http.StatusNoContent)

	rc.base.WSCurrentUpdate()
}

// UploadRunJSON will take a json and import the runs
func (rc *RunController) UploadRunJSON(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var runs []models.Run

	err := json.NewDecoder(r.Body).Decode(&runs)
	if err != nil {
		rc.base.Response("", "couldn't unmarshal body", http.StatusInternalServerError, w)
		log.Printf("Error in UploadRunJSON: %v", err)
		return
	}

	rc.base.MGS.DB("marathon").C("runs").RemoveAll(nil)

	for _, run := range runs {
		run.RunID = bson.NewObjectId()
		err := rc.base.MGS.DB("marathon").C("runs").Insert(run)
		if err != nil {
			panic("error adding run from UploadRunJSON into db")
		}
	}

	log.Printf("imported %v runs", len(runs))
	w.WriteHeader(http.StatusNoContent)
}
func (rc *RunController) checkForUpdate() {
	go func() {
		var res []byte
		var ts social.TwitchSettings
		res, err := rc.base.RedisClient.Get("twitchSettings").Bytes()
		if err != nil {
			if err == redis.Nil {
				return
			}
			rc.base.LogError("error while getting twitch settings from redis", err, true)
			return
		}

		json.Unmarshal(res, &ts)

		if ts.Update {
			rc.base.SocialUpdatesChan <- 1
		}

	}()

	go func() {
		res, err := rc.base.RedisClient.Get("twitterSettings").Bytes()
		if err != nil {
			if err == redis.Nil {
				return
			}
			rc.base.LogError("error while getting settings from twitter", err, true)
			return
		}
		if b, err := strconv.ParseBool(string(res)); err == nil && b {
			rc.base.SocialUpdatesChan <- 2
		}
	}()

	go func() {
		rc.base.SocialUpdatesChan <- 0x50
	}()
}
