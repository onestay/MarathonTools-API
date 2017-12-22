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

	"github.com/go-redis/redis"

	"github.com/onestay/MarathonTools-API/api/models"

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

type twitchTitleOptions struct {
	Game     string
	Runner   []models.PlayerInfo
	Platform string
	Estimate string
	Category string
}

const (
	authorizeURL     = "https://api.twitch.tv/kraken/oauth2/authorize"
	tokenURL         = "https://api.twitch.tv/kraken/oauth2/token"
	revokeURL        = "https://api.twitch.tv/kraken/oauth2/revoke"
	channelURL       = "https://api.twitch.tv/kraken/channel"
	updateChannelURL = "https://api.twitch.tv/kraken/channels"
)

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

	title := sc.twitchExecuteTemplate()
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

func (sc Controller) TwitchExecuteTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := sc.twitchExecuteTemplate()
	if res == "ERROR" {
		sc.base.Response("", res, http.StatusInternalServerError, w)

	}
	sc.base.Response(res, "", http.StatusOK, w)
}

func (sc Controller) twitchExecuteTemplate() string {
	currentRun := sc.base.CurrentRun
	c := twitchTitleOptions{currentRun.GameInfo.GameName, currentRun.Players, currentRun.RunInfo.Platform, currentRun.RunInfo.Estimate, currentRun.RunInfo.Category}

	res, err := sc.base.RedisClient.Get("twitchSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			return "ERROR"
		}
		sc.base.LogError("error while getting twitch settings from redis", err, true)
		return "ERROR"
	}

	ts := TwitchSettings{}

	json.Unmarshal(res, &ts)

	tmpl, err := template.New("run").Parse(ts.TemplateString)

	var execTemplate bytes.Buffer
	err = tmpl.Execute(&execTemplate, c)
	if err != nil {
		sc.base.LogError("while executing template", err, true)
		return "ERROR"
	}

	return execTemplate.String()
}

type TwitchSettings struct {
	TitleUpdate    bool   `json:"titleUpdate"`
	GameUpdate     bool   `json:"gameUpdate"`
	Viewers        bool   `json:"viewers"`
	TemplateString string `json:"templateString"`
}

func (sc Controller) TwitchSetSettings(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var ts TwitchSettings
	err := json.NewDecoder(r.Body).Decode(&ts)
	if err != nil {
		sc.base.LogError("while decoding the body for setting save", err, true)
		return
	}

	ser, err := json.Marshal(ts)
	if err != nil {
		sc.base.LogError("while marshal the body for setting save", err, true)
		return
	}

	sc.base.RedisClient.Set("twitchSettings", ser, 0)

	w.Header().Add("Content-Type", "application/json")

	w.Write(ser)
}

func (sc Controller) TwitchGetSettings(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var res []byte

	res, err := sc.base.RedisClient.Get("twitchSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			sc.base.Response("", "no settings have been saved", 404, w)
			return
		}
		sc.base.LogError("error while getting twitch settings from redis", err, true)
		return
	}
	w.Header().Add("Content-Type", "application/json")

	w.Write(res)
}
