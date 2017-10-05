package runs

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/onestay/MarathonTools-API/api/models"
	"gopkg.in/mgo.v2/bson"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
)

// RunController contains all the methods needed to control runs
type RunController struct {
	base *common.Controller
}

// NewRunController returns a new run controller
func NewRunController(b *common.Controller) *RunController {
	return &RunController{
		base: b,
	}
}

// AddRun will add a run to the database
func (rc RunController) AddRun(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	run := models.Run{}
	json.NewDecoder(r.Body).Decode(&run)

	run.RunID = bson.NewObjectId()

	err := rc.base.MGS.DB("marathon").C("runs").Insert(run)
	if err != nil {
		rc.base.Response("", "err adding run", http.StatusInternalServerError, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetRuns will return all runs from the mgo collection
func (rc RunController) GetRuns(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	runs := []models.Run{}

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

// GetRun will return a single run based on objectid
func (rc RunController) GetRun(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

func (rc RunController) DeleteRun(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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
}

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
}
