package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/onestay/MarathonTools-API/api/routes/donations"

	"github.com/onestay/MarathonTools-API/api/donationProviders"
	"github.com/onestay/MarathonTools-API/api/models"
	"github.com/onestay/MarathonTools-API/api/routes/timer"

	"github.com/go-redis/redis"

	"github.com/onestay/MarathonTools-API/api/routes/social"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
	"github.com/onestay/MarathonTools-API/api/routes/runs"
	"github.com/onestay/MarathonTools-API/ws"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var (
	mgs                *mgo.Session
	redisClient        *redis.Client
	port               string
	twitchClientID     string
	twitchClientSecret string
	twitchCallback     string
	twitterKey         string
	twitterSecret      string
	twitterCallback    string
	refreshInterval    int
	marathonSlug       string
)

type Server struct {
	r *httprouter.Router
}

func init() {
	mgs = getMongoSession()
	redisClient = getRedisClient()

	// parse env vars
	twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	twitchCallback = os.Getenv("TWITCH_CALLBACK")
	twitterKey = os.Getenv("TWITTER_KEY")
	twitterSecret = os.Getenv("TWITTER_SECRET")
	twitterCallback = os.Getenv("TWITTER_CALLBACK")
	marathonSlug = os.Getenv("MARATHON_SLUG")
	i, err := strconv.Atoi(os.Getenv("REFRESH_INTERVAL"))
	if err != nil {
		panic("REFRESH_INTERVAL has to be a valid int in ms")
	}
	refreshInterval = i
	port = os.Getenv("HTTP_PORT")
}

func main() {
	importRuns()
	startHTTPServer()
	// os.Getenv("jsonruns") == "true"
}

func startHTTPServer() {
	r := httprouter.New()
	hub := ws.NewHub()
	baseController := common.NewController(hub, mgs, 0, redisClient)
	socialController := social.NewSocialController(twitchClientID, twitchClientSecret, twitterCallback, twitterKey, twitterSecret, twitterCallback, baseController)
	timeController := timer.NewTimeController(baseController, refreshInterval)
	runController := runs.NewRunController(baseController)

	srDonationProvider, err := donationProviders.NewSRComDonationProvider(marathonSlug)
	if err != nil {
		panic(err)
	}

	donationController := donations.NewDonationController(baseController, srDonationProvider)
	go hub.Run()

	r.GET("/ws", func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ws.ServeWs(hub, w, r, baseController.SendInitialData())
	})

	// routes for donation endpoint
	r.GET("/donations/total", donationController.GetTotal)
	r.GET("/donations/all", donationController.GetAll)
	r.GET("/donations/total/amount", donationController.GetTotalDonations)
	r.GET("/donations/total/update/start", donationController.StartTotalUpdate)
	r.GET("/donations/total/update/stop", donationController.StopTotalUpdate)
	// routes for run endpoint
	r.GET("/run/get/all", runController.GetRuns)
	r.GET("/run/get/single/:id", runController.GetRun)
	r.GET("/run/get/active", runController.ActiveRuns)

	r.DELETE("/run/delete/:id", runController.DeleteRun)

	r.PATCH("/run/update/:id", runController.UpdateRun)

	r.POST("/run/move/:id/:after", runController.MoveRun)
	r.POST("/run/add/single", runController.AddRun)
	r.POST("/run/switch", runController.SwitchRun)

	// social stuff
	r.GET("/social/twitch/oauthurl", socialController.TwitchOAuthURL)
	r.GET("/social/twitch/verify", socialController.TwitchCheckForAuth)
	r.POST("/social/twitch/auth", socialController.TwitchGetToken)
	r.DELETE("/social/twitch/token", socialController.TwitchDeleteToken)
	r.GET("/social/twitch/executetemplate", socialController.TwitchExecuteTemplate)
	r.PUT("/social/twitch/update", socialController.TwitchUpdateInfo)
	r.PUT("/social/twitch/settings", socialController.TwitchSetSettings)
	r.GET("/social/twitch/settings", socialController.TwitchGetSettings)

	r.GET("/social/twitter/oauthurl", socialController.TwitterOAuthURL)
	r.GET("/social/twitter/verify", socialController.TwitterCheckForAuth)
	r.POST("/social/twitter/auth", socialController.TwitterCallback)
	r.DELETE("/social/twitter/token", socialController.TwitterDeleteToken)
	r.POST("/social/twitter/update", socialController.TwitterSendUpdate)
	r.POST("/social/twitter/template", socialController.TwitterAddTemplate)
	r.GET("/social/twitter/template", socialController.TwitterGetTemplates)
	r.DELETE("/social/twitter/template/:index", socialController.TwitterDeleteTemplate)
	r.PUT("/social/twitter/settings", socialController.TwitterSetSettings)
	r.GET("/social/twitter/settings", socialController.TwitterGetSettings)

	// timer stuff
	r.POST("/timer/start", timeController.TimerStart)
	r.POST("/timer/pause", timeController.TimerPause)
	r.POST("/timer/resume", timeController.TimerResume)
	r.POST("/timer/finish", timeController.TimerFinish)
	r.POST("/timer/player/finish/:id", timeController.TimerPlayerFinish)
	r.POST("/timer/reset", timeController.TimerReset)

	log.Println("server running on " + port)
	log.Fatal(http.ListenAndServe(port, &Server{r}))
}

func getMongoSession() *mgo.Session {
	s, err := mgo.Dial("mongodb://mongo")
	if err != nil {
		panic("Couldn't establish mgo session " + err.Error())
	}

	return s
}

func getRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "redis:6379",
		Password: "",
		DB:       0,
	})

	return client
}

func importRuns() {
	runs := []models.Run{}
	runFile, err := os.Open("./config/runs.json")
	if err != nil {
		log.Println("no runs file... this can be ignored if runs are already imported")
		return
	}

	json.NewDecoder(runFile).Decode(&runs)

	for _, run := range runs {
		run.RunID = bson.NewObjectId()
		err := mgs.DB("marathon").C("runs").Insert(run)
		if err != nil {
			panic("error adding run from json into db")
		}
	}
	log.Printf("imported %v runs", len(runs))
	err = os.Rename("./config/runs.json", "./config/runs_imported.json")
	if err != nil {
		log.Println("error renaming runs file. Please rename manually so runs aren't imported on restart")
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, PUT, PATCH, OPTIONS, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		// idk why but because of some reason 504 was the answer to every pre flight request
		// so this hack has to work
		w.WriteHeader(200)
	} else {
		s.r.ServeHTTP(w, r)
	}
}
