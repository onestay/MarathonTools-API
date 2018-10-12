package social

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"text/template"
	"time"

	"github.com/go-redis/redis"

	"github.com/onestay/MarathonTools-API/api/models"

	"github.com/julienschmidt/httprouter"
)

// TODO: add the channel id to the twitch settings so user can specify channel id. Defaults to authenticated user channel id

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

// this will mostly use the old twitch api since most of the endpoints I need aren't available in the new one
// I will try to use the new one as much as possible tho
const (
	authorizeURL      = "https://api.twitch.tv/kraken/oauth2/authorize"
	tokenURL          = "https://api.twitch.tv/kraken/oauth2/token"
	revokeURL         = "https://api.twitch.tv/kraken/oauth2/revoke"
	channelURL        = "https://api.twitch.tv/kraken/channel"
	updateChannelURL  = "https://api.twitch.tv/kraken/channels"
	getStreamURL      = "https://api.twitch.tv/helix/streams"
	refreshTokenURL   = "https://id.twitch.tv/oauth2/token"
	playCommercialURL = "https://api.twitch.tv/kraken/channels"
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

	var resp *http.Response

	resp, err = client.Do(req)
	if err != nil {
		log.Printf("Error doing request. Err: %v", err)
	}

	if resp.StatusCode == 400 {
		token, err := sc.twitchRefreshToken()
		if err != nil {
			sc.base.LogError("while trying to get refresh token", err, true)
			return
		}
		req, err := http.NewRequest("GET", channelURL, nil)
		if err != nil {
			log.Println("Error creating request to get Channel ID")
		}

		req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
		req.Header.Add("Authorization", "OAuth "+token)
		req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")

		var resp *http.Response

		resp, err = client.Do(req)
		if err != nil {
			log.Printf("Error doing request. Err: %v", err)
		}

		if resp.StatusCode == 400 {
			sc.base.LogError("couldn't get channel id even after successfull refresh token refresh. Bad auth", err, true)
			return
		}
	}

	id := struct {
		ID string `json:"_id"`
	}{}

	json.NewDecoder(resp.Body).Decode(&id)
	t.ChannelID = id.ID

	res <- true
}

func (sc Controller) twitchUpdateInfo() error {
	client := http.Client{}

	b, err := sc.base.RedisClient.Get("twitchAuth").Bytes()
	if err != nil {
		if err == redis.Nil {
			return errors.New("twitch updates enabled but no login data saved")
		}
		return err
	}
	t := TwitchResponse{}

	json.Unmarshal(b, &t)

	title := sc.twitchExecuteTemplate()
	game := sc.base.CurrentRun.GameInfo.GameName

	uri, err := url.Parse(updateChannelURL + "/" + t.ChannelID)
	if err != nil {
		return err
	}

	type channel struct {
		Game   string `json:"game,omitempty"`
		Status string `json:"status,omitempty"`
	}

	type Payload struct {
		Channel channel `json:"channel,omitempty"`
	}

	var ch channel

	ts, err := sc.twitchGetSettings()
	if err != nil {
		return err
	}

	if ts.GameUpdate && ts.TitleUpdate {
		ch = channel{
			Status: title,
			Game:   game,
		}
	} else if ts.GameUpdate && !ts.TitleUpdate {
		ch = channel{
			Game: game,
		}
	} else if ts.TitleUpdate && !ts.GameUpdate {
		ch = channel{
			Status: title,
		}
	} else {
		// in that case neither run nor title update are enabled.
		return nil
	}
	payload := Payload{ch}

	result, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", uri.String(), bytes.NewReader(result))
	if err != nil {
		return err
	}

	req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
	req.Header.Add("Authorization", "OAuth "+t.AccessToken)
	req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		token, err := sc.twitchRefreshToken()
		if err != nil {
			return fmt.Errorf("error while trying to get refresh token: %v", err)
		}

		req, err := http.NewRequest("PUT", uri.String(), bytes.NewReader(result))
		if err != nil {
			return err
		}

		req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
		req.Header.Add("Authorization", "OAuth "+token)
		req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
		req.Header.Add("Content-Type", "application/json")

		res, err := client.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode != 200 {
			return fmt.Errorf("couldn't update info even afer getting refreshtoken. Status code is %v", res.StatusCode)
		}
	}

	return nil
}

// TwitchUpdateInfo will update the game and title for the connected twitch account
func (sc Controller) TwitchUpdateInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := sc.twitchUpdateInfo()
	if err != nil {
		sc.base.Response("", "error sending twitter update", http.StatusInternalServerError, w)
	}

	w.WriteHeader(http.StatusNoContent)
}

// TwitchExecuteTemplate will execute the template string given via config
func (sc Controller) TwitchExecuteTemplate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := sc.twitchExecuteTemplate()
	if res == "NOTEMPLATE" {
		sc.base.Response("", "NOTEMPLATE", 200, w)
		return
	} else if res == "ERROR" {
		sc.base.Response("", res, http.StatusInternalServerError, w)
		return

	}
	sc.base.Response(res, "", http.StatusOK, w)
}

func (sc Controller) twitchExecuteTemplate() string {
	currentRun := sc.base.CurrentRun
	c := twitchTitleOptions{currentRun.GameInfo.GameName, currentRun.Players, currentRun.RunInfo.Platform, currentRun.RunInfo.Estimate, currentRun.RunInfo.Category}

	res, err := sc.base.RedisClient.Get("twitchSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			return "NOTEMPLATE"
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

// TwitchSettings defines the settings for twitch integration
type TwitchSettings struct {
	TitleUpdate    bool   `json:"titleUpdate"`
	GameUpdate     bool   `json:"gameUpdate"`
	Viewers        bool   `json:"viewers"`
	TemplateString string `json:"templateString"`
}

// TwitchSetSettings sets the settings
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

	go func() {
		if ts.Viewers {
			sc.twitchStartViewerUpdates()
		} else {
			sc.twitchStopViewerUpdates()
		}
	}()
	sc.base.RedisClient.Set("twitchSettings", ser, 0)

	w.Header().Add("Content-Type", "application/json")

	w.Write(ser)
}

// TwitchGetSettings returns settings
func (sc Controller) TwitchGetSettings(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	var res []byte

	res, err := sc.base.RedisClient.Get("twitchSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			sc.base.Response("", "no settings have been saved", 200, w)
			return
		}
		sc.base.LogError("error while getting twitch settings from redis", err, true)
		return
	}
	w.Header().Add("Content-Type", "application/json")

	w.Write(res)
}

func (sc Controller) twitchGetSettings() (*TwitchSettings, error) {
	var res []byte
	var ts TwitchSettings
	res, err := sc.base.RedisClient.Get("twitchSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.New("no Twitch settings saved")
		}
		return nil, err
	}

	json.Unmarshal(res, &ts)

	return &ts, nil
}

var viewerUpdateTicker *time.Ticker

func (sc Controller) twitchStartViewerUpdates() {
	viewerUpdateTicker = time.NewTicker(1 * time.Minute)

	data := struct {
		DataType string `json:"dataType"`
		Viewers  int    `json:"viewers"`
	}{"twitchViewerUpdate", sc.getTwitchViewers()}

	d, _ := json.Marshal(data)

	sc.base.WS.Broadcast <- d

	go func() {
		for {
			select {
			case <-viewerUpdateTicker.C:
				data := struct {
					DataType string `json:"dataType"`
					Viewers  int    `json:"viewers"`
				}{"twitchViewerUpdate", sc.getTwitchViewers()}

				d, _ := json.Marshal(data)

				sc.base.WS.Broadcast <- d
			}
		}
	}()
}

func (sc Controller) twitchStopViewerUpdates() {
	if viewerUpdateTicker != nil {
		go func() {
			data := struct {
				DataType string `json:"dataType"`
				Viewers  int    `json:"viewers"`
			}{"twitchViewerUpdate", -1}

			d, _ := json.Marshal(data)

			sc.base.WS.Broadcast <- d
		}()
		viewerUpdateTicker.Stop()

		viewerUpdateTicker = nil
	}
}

func (sc Controller) getTwitchViewers() int {
	b, _ := sc.base.RedisClient.Get("twitchAuth").Bytes()
	t := TwitchResponse{}

	json.Unmarshal(b, &t)

	req, _ := http.NewRequest("GET", getStreamURL+"?user_id="+t.ChannelID, nil)
	req.Header.Add("Client-ID", sc.twitchInfo.ClientID)

	res, _ := sc.base.HTTPClient.Do(req)

	twitchRes := struct {
		Data []struct {
			ViewerCount int `json:"viewer_count"`
		} `json:"data"`
	}{}

	json.NewDecoder(res.Body).Decode(&twitchRes)
	if len(twitchRes.Data) != 0 {
		return twitchRes.Data[0].ViewerCount
	}
	return -1

}

func (sc Controller) TwitchPlayCommercial(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	b, err := sc.base.RedisClient.Get("twitchAuth").Bytes()
	if err != nil {
		if err == redis.Nil {
			sc.base.Response("", "no twitch auth saved", http.StatusBadRequest, w)
			return
		}
		sc.base.Response("", "couldn't get twitch auth info", http.StatusInternalServerError, w)
		return
	}
	t := TwitchResponse{}

	json.Unmarshal(b, &t)

	commercialTimes := map[int]bool{30: true, 60: true, 90: true, 120: true, 150: true, 180: true}

	body := struct {
		Length int `json:"length"`
	}{}

	json.NewDecoder(r.Body).Decode(&body)

	bRes, _ := json.Marshal(body)

	defer r.Body.Close()

	if commercialTimes[body.Length] {

		req, _ := http.NewRequest("POST", playCommercialURL+"/"+t.ChannelID+"/commercial", bytes.NewBuffer(bRes))

		req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
		req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
		req.Header.Add("Authorization", "OAuth "+t.AccessToken)
		req.Header.Add("Content-Type", "application/json")

		res, _ := sc.base.HTTPClient.Do(req)

		if res.StatusCode != 200 {
			// as documented at https://dev.twitch.tv/docs/v5/reference/channels/#start-channel-commercial
			if res.StatusCode == 422 {
				sc.base.Response("", "invalid length or latest commercial less than 8 minutes or channel is not twitch partner", http.StatusUnprocessableEntity, w)
				return
			}

			// get refresh access token and redo request
			token, err := sc.twitchRefreshToken()
			if err != nil {
				sc.base.Response("", "error while getting twitch refresh token", http.StatusInternalServerError, w)
			}

			req, _ := http.NewRequest("POST", playCommercialURL+"/"+t.ChannelID+"/commercial", bytes.NewBuffer(bRes))

			req.Header.Add("Accept", "application/vnd.twitchtv.v5+json")
			req.Header.Add("Client-ID", sc.twitchInfo.ClientID)
			req.Header.Add("Authorization", "OAuth "+token)
			req.Header.Add("Content-Type", "application/json")

			res, _ := sc.base.HTTPClient.Do(req)
			if res.StatusCode != 200 {
				sc.base.Response("", "an error occured playing the commercial after getting refresh token. twitch status code is "+res.Status, http.StatusBadRequest, w)
				return

			}
		}
		sc.base.Response("ok", "", http.StatusOK, w)
	} else {
		sc.base.Response("", "invalid commerical time", http.StatusBadRequest, w)
	}

}
