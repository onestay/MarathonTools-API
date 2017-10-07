package main

import (
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/api/routes/runs"
	"github.com/onestay/MarathonTools-API/ws"
	"gopkg.in/mgo.v2"
)

var (
	mgs      *mgo.Session
	marathon *models.Marathon
)

func init() {
	mgs = getSession()
}

func main() {
	startHTTPServer()
	/* if os.Getenv("jsonruns") == "true" {
		runs := []models.Run{}
		runFile, err := os.Open("./config/runs.json")
		if err != nil {
			panic("jsonruns set to true but no run file provided")
		}

		json.NewDecoder(runFile).Decode(&runs)

		for _, run := range runs {
			run.RunID = bson.NewObjectId()
			fmt.Println(run)
			err := mgs.DB("marathon").C("runs").Insert(run)
			if err != nil {
				panic("error adding run from json into db")
			}
		}
		os.Rename("./config/runs.json", "./config/runs_imported.json")
	} */
}

func startHTTPServer() {
	r := httprouter.New()
	hub := ws.NewHub()
	baseController := common.NewController(hub, mgs, 0)
	rc := runs.NewRunController(baseController)
	go hub.Run()

	r.GET("/ws", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ws.ServeWs(hub, w, r)
	})

	r.GET("/run/get/all", rc.GetRuns)
	r.GET("/run/get/single/:id", rc.GetRun)
	r.GET("/run/get/active", rc.ActiveRuns)
	r.DELETE("/run/delete/:id", rc.DeleteRun)
	r.PATCH("/run/update/:id", rc.UpdateRun)
	r.POST("/run/move/:id/:after", rc.MoveRun)
	r.POST("/run/add/single", rc.AddRun)

	log.Println("server running on :3001")
	log.Fatal(http.ListenAndServe(":3001", r))
}

func getSession() *mgo.Session {
	s, err := mgo.Dial("mongodb://mongo")
	if err != nil {
		panic("Couldn't establish mgo session " + err.Error())
	}

	return s
}
