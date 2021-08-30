package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/onestay/MarathonTools-API/api/routes/donations"

	"github.com/onestay/MarathonTools-API/api/donationProviders"
	"github.com/onestay/MarathonTools-API/api/routes/timer"

	"github.com/go-redis/redis"

	"github.com/joho/godotenv"
	"github.com/onestay/MarathonTools-API/api/routes/social"

	"github.com/julienschmidt/httprouter"
	"github.com/onestay/MarathonTools-API/api/common"
	"github.com/onestay/MarathonTools-API/api/routes/runs"
	"github.com/onestay/MarathonTools-API/ws"
	mgo "gopkg.in/mgo.v2"
)

var (
	mgs                                          *mgo.Session
	redisClient                                  *redis.Client
	port                                         string
	twitchClientID                               string
	twitchClientSecret                           string
	twitchCallback                               string
	twitterKey                                   string
	twitterSecret                                string
	twitterCallback                              string
	refreshInterval                              int
	marathonSlug                                 string
	gdqURL, gdqEventID, gdqUsername, gdqPassword string
	mgoURL, redisURL                             string
)

type Server struct {
	r *httprouter.Router
}

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file.")
	}
	parseEnvVars()
	log.Printf("Connecting to mgo server at %v", mgoURL)
	mgs = getMongoSession()
	log.Printf("Connecting to redis server at %v", redisURL)
	redisClient = getRedisClient()
}

func main() {
	startHTTPServer()
}

func startHTTPServer() {
	r := httprouter.New()
	hub := ws.NewHub()
	log.Println("Initializing base controller...")
	baseController := common.NewController(hub, mgs, 0, redisClient)
	log.Println("Initializing social controller...")
	social.NewSocialController(twitchClientID, twitchClientSecret, twitchCallback, twitterKey, twitterSecret, twitterCallback, baseController, r)
	log.Println("Initializing time controller...")
	timer.NewTimeController(baseController, refreshInterval, r)
	log.Println("Initializing run controller")
	runs.NewRunController(baseController, r)

	var donProv donations.DonationProvider
	donationsEnabled := true
	var err error

	if os.Getenv("DONATION_PROVIDER") == "gdq" {
		log.Println("Creating new GDQ donation provider")
		donProv, err = donationProviders.NewGDQDonationProvider(gdqURL, gdqEventID, gdqUsername, gdqPassword)
		if err != nil {
			log.Printf("Error during gdq donation provider creation: %v", err)
			donationsEnabled = false
		}
	} else if os.Getenv("DONATION_PROVIDER") == "srcom" {
		log.Println("Creating new speedrun.com donation provider")
		donProv, err = donationProviders.NewSRComDonationProvider(marathonSlug)
		if err != nil {
			log.Printf("Error during donation provider creation: %v", err)
			donationsEnabled = false
		}
	} else {
		log.Print("No donation provider specified")
		donationsEnabled = false
	}

	donationController := donations.NewDonationController(baseController, donProv, donationsEnabled)

	log.Println("Starting websocket hub...")
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

	// checklist stuff
	r.POST("/checklist/add", baseController.CL.AddItem)
	r.DELETE("/checklist/delete", baseController.CL.DeleteItem)
	r.PUT("/checklist/toggle", baseController.CL.ToggleItem)
	r.GET("/checklist/done", baseController.CL.CheckDoneHTTP)
	r.GET("/checklist", baseController.CL.GetChecklist)

	// settings stuff
	r.POST("/settings", baseController.Settings.SetSettings)
	r.GET("/settings", baseController.Settings.GetSettings)

	log.Println("server running on " + port)
	log.Fatal(http.ListenAndServe(port, &Server{r}))
}

func getMongoSession() *mgo.Session {
	s, err := mgo.Dial(mgoURL)
	if err != nil {
		panic("Couldn't establish mgo session " + err.Error())
	}

	return s
}

func getRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     redisURL + ":6379",
		Password: "",
		DB:       0,
	})

	return client
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

func parseEnvVars() {
	mgoURL = os.Getenv("MONGO_SERVER")
	redisURL = os.Getenv("REDIS_SERVER")
	twitchClientID = os.Getenv("TWITCH_CLIENT_ID")
	twitchClientSecret = os.Getenv("TWITCH_CLIENT_SECRET")
	twitchCallback = os.Getenv("TWITCH_CALLBACK")
	twitterKey = os.Getenv("TWITTER_KEY")
	twitterSecret = os.Getenv("TWITTER_SECRET")
	twitterCallback = os.Getenv("TWITTER_CALLBACK")
	i, err := strconv.Atoi(os.Getenv("REFRESH_INTERVAL"))
	if err != nil && len(os.Getenv("REFRESH_INTERVAL")) != 0 {
		log.Println("Error parsing REFRESH_INTERVAL defaulting to 100ms")
		i = 100
	} else if err != nil {
		i = 100
	}
	refreshInterval = i
	port = os.Getenv("HTTP_PORT")
	if len(port) == 0 {
		port = ":3000"
	}

	if os.Getenv("DONATION_PROVIDER") == "gdq" {
		gdqURL = os.Getenv("GDQ_TRACKER_URL")
		gdqEventID = os.Getenv("GDQ_TRACKER_EVENT_ID")
		gdqUsername = os.Getenv("GDQ_TRACKER_USERNAME")
		gdqPassword = os.Getenv("GDQ_TRACKER_PASSWORD")
	} else if os.Getenv("DONATION_PRODIVER") == "srcom" {
		marathonSlug = os.Getenv("MARATHON_SLUG")
	} else if len(os.Getenv("DONATION_PROVIDER")) == 0 {
		log.Println("No donations provider specified. Donations disabled.")
	} else {
		log.Printf("Unknown donation provider %v. Donations disabled.", os.Getenv("DONATION_PROVIDER"))
	}

}
