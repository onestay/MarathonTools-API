package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/models"
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
	log.SetFlags(log.LstdFlags)
	startHTTPServer()
	ensureMarathon()
}

func startHTTPServer() {
	r := httprouter.New()
	hub := ws.NewHub()
	// baseController := common.NewController(hub)
	// rc := runs.NewRunController(baseController)
	go hub.Run()

	r.GET("/ws", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ws.ServeWs(hub, w, r)
	})
	log.Println("server running on :5123")
	log.Fatal(http.ListenAndServe(":5123", r))

}

func ensureMarathon() bool {
	names, err := mgs.DatabaseNames()
	if err != nil {
		panic(err)
	}
	log.Println("Hi")
	for _, n := range names {
		fmt.Println(n)
	}
	return true
}

func getSession() *mgo.Session {
	s, err := mgo.Dial("mongodb://mongo")
	if err != nil {
		panic("Couldn't establish mgo session " + err.Error())
	}

	return s
}
