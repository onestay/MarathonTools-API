package social

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/onestay/MarathonTools-API/api/models"

	"github.com/go-redis/redis"

	"github.com/julienschmidt/httprouter"
)

// TwitchResponse is the response returned from the twitch servers for a access token
type TwitchResponse struct {
	AccessToken  string `json:"access_token" bson:"accessToken"`
	RefreshToken string `json:"refresh_token" bson:"refreshToken"`
	ExpiresIn    int    `json:"expires_in" bson:"expiresIn"`
	Scope        string `json:"scope" bson:"scope"`
	InsertDate   time.Time
	ChannelID    string
}

type channelError struct {
	Error        bool
	ErrorMessage string
}

const (
	authorizeURL     = "https://api.twitch.tv/kraken/oauth2/authorize"
	tokenURL         = "https://api.twitch.tv/kraken/oauth2/token"
	revokeURL        = "https://api.twitch.tv/kraken/oauth2/revoke"
	channelURL       = "https://api.twitch.tv/kraken/channel"
	updateChannelURL = "https://api.twitch.tv/kraken/channels"
)

// TwitchOAuthURL will return the oauth url used for twitch auth
func (sc Controller) TwitchOAuthURL(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	var uri *url.URL
	uri, _ = url.Parse(authorizeURL)

	parameters := url.Values{}
	parameters.Add("client_id", sc.twitchInfo.ClientID)
	parameters.Add("redirect_uri", sc.twitchInfo.RedirectURI)
	parameters.Add("response_type", "code")
	parameters.Add("scope", sc.twitchInfo.Scope)
	uri.RawQuery = parameters.Encode()

	sc.base.Response(uri.String(), "", http.StatusOK, w)

}

// TwitchGetToken will return an access token from the twitch servers after a code from the client has been obtained
func (sc Controller) TwitchGetToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	code := r.URL.Query().Get("code")
	var uri *url.URL

	uri, _ = url.Parse(tokenURL)

	parameters := url.Values{}
	parameters.Add("client_id", sc.twitchInfo.ClientID)
	parameters.Add("redirect_uri", sc.twitchInfo.RedirectURI)
	parameters.Add("client_secret", sc.twitchInfo.ClientSecret)
	parameters.Add("grant_type", "authorization_code")
	parameters.Add("code", code)
	uri.RawQuery = parameters.Encode()

	res, err := http.Post(uri.String(), "", nil)
	if err != nil || res.StatusCode != 200 {
		log.Printf("Error in getting oauth token, err: %v", err)
		sc.base.Response("", "error getting oauth token", 500, w)
		return
	}

	resStruct := TwitchResponse{}
	resStruct.InsertDate = time.Now()
	json.NewDecoder(res.Body).Decode(&resStruct)

	resChan := make(chan bool)

	go sc.getChannelID(resChan, &resStruct)
	<-resChan

	b, _ := json.Marshal(resStruct)

	err = sc.base.RedisClient.Set("twitchAuth", b, 0).Err()
	if err != nil {
		log.Printf("Error in setting auth info, err: %v", err)
		sc.base.Response("", "error", 500, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TwitchCheckForAuth will check if there is an access token available. It doesn't necessairly say if it's expired or invalid
func (sc Controller) TwitchCheckForAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, err := sc.base.RedisClient.Get("twitchAuth").Result()
	if err == redis.Nil {
		sc.base.Response("false", "", 200, w)
		return
	}

	sc.base.Response("true", "", 200, w)
}

// TwitchDeleteToken will delete and revoke the twitch token
func (sc Controller) TwitchDeleteToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b, _ := sc.base.RedisClient.Get("twitchAuth").Bytes()
	t := TwitchResponse{}
	var uri *url.URL

	json.Unmarshal(b, &t)

	uri, _ = url.Parse(revokeURL)

	parameters := url.Values{}
	parameters.Add("token", t.AccessToken)
	uri.RawQuery = parameters.Encode()

	http.Post(uri.String(), "", nil)

	sc.base.RedisClient.Del("twitchAuth")

	w.WriteHeader(http.StatusNoContent)
}

func (sc Controller) getChannelID(res chan bool, t *TwitchResponse) {
	client := http.Client{}

	req, err := http.NewRequest("GET", channelURL, nil)
	if err != nil {
		log.Println("Error creating request to get Channel ID")
	}

	req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
	req.Header.Add("Authorization", "OAuth "+t.AccessToken)
	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error doing request. Err: %v", err)
	}

	id := struct {
		ID string `json:"_id"`
	}{}

	json.NewDecoder(resp.Body).Decode(&id)
	t.ChannelID = id.ID

	res <- true
}

// TwitchUpdateInfo will update the game and title for the connected twitch account
func (sc Controller) TwitchUpdateInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	client := http.Client{}

	b, err := sc.base.RedisClient.Get("twitchAuth").Bytes()
	if err != nil {
		sc.base.LogError("getting twitch auth info from redis", err, true)
	}
	t := TwitchResponse{}

	json.Unmarshal(b, &t)

	title := sc.twitchParseTemplate()
	game := sc.base.CurrentRun.GameInfo.GameName

	uri, err := url.Parse(updateChannelURL + "/" + t.ChannelID)
	if err != nil {
		sc.base.LogError("parsing channel url", err, true)
		return
	}
	type channel struct {
		Game   string `json:"game"`
		Status string `json:"status"`
	}

	type Payload struct {
		Channel channel `json:"channel,omitempty"`
	}

	ch := channel{game, title}
	payload := Payload{ch}

	result, err := json.Marshal(payload)
	if err != nil {
		sc.base.LogError("Could not marshall json for twitch update", err, true)
	}

	req, err := http.NewRequest("PUT", uri.String(), bytes.NewReader(result))
	if err != nil {
		sc.base.LogError("creating request to update twitch info", err, true)
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Authorization", "OAuth "+t.AccessToken)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		sc.base.LogError("sending request to update game info", err, true)
	}

	if res.StatusCode != 200 {
		sc.base.LogError("twitch api response was not 200", errors.New(res.Status), true)
	}
}

func (sc Controller) twitchParseTemplate() string {
	currentRun := sc.base.CurrentRun
	c := twitchTitleOptions{currentRun.GameInfo.GameName, currentRun.Players, currentRun.RunInfo.Platform, currentRun.RunInfo.Estimate, currentRun.RunInfo.Category}

	templateString, err := sc.base.RedisClient.Get("twitchTemplate").Result()
	if err != nil {
		sc.base.LogError("getting the template string from redis. Make sure the Twitch title template is set.", err, true)
		return "ERROR"
	}

	tmpl, err := template.New("run").Parse(templateString)

	var res bytes.Buffer
	err = tmpl.Execute(&res, c)
	if err != nil {
		sc.base.LogError("while executing template", err, true)
		return "ERROR"
	}

	return res.String()
}

type twitchTitleOptions struct {
	Game     string
	Runner   []models.PlayerInfo
	Platform string
	Estimate string
	Category string
}

// TwitchTitleTemplate will set the redis entry for the twitch title template
func (sc Controller) TwitchTitleTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	templateString := struct {
		TemplateString string `json:"template"`
	}{}

	json.NewDecoder(r.Body).Decode(&templateString)

	err := sc.base.RedisClient.Set("twitchTemplate", templateString.TemplateString, 0).Err()
	if err != nil {
		sc.base.LogError("while saving the template", err, true)
	}

	w.WriteHeader(http.StatusNoContent)
}
