package social

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"text/template"
	"time"

	"github.com/go-redis/redis"

	"github.com/onestay/MarathonTools-API/api/models"

	"github.com/julienschmidt/httprouter"
)

// TODO: add the channel id to the twitch settings so user can specify channel id. Defaults to authenticated user channel id

// TwitchResponse is the response returned from the twitch servers for a access token
type TwitchResponse struct {
	AccessToken  string   `json:"access_token" bson:"accessToken"`
	RefreshToken string   `json:"refresh_token" bson:"refreshToken"`
	ExpiresIn    int      `json:"expires_in" bson:"expiresIn"`
	Scope        []string `json:"scope" bson:"scope"`
	InsertDate   time.Time
	ChannelID    string
}

type twitchTitleOptions struct {
	Game     string
	Runner   []models.PlayerInfo
	Platform string
	Estimate string
	Category string
}

func (sc Controller) twitchUpdateInfo() error {
	client := http.Client{}

	title := sc.twitchExecuteTemplate()
	game := sc.base.CurrentRun.GameInfo.GameName

	uri, err := url.Parse(sc.socialAuth.url + "/api/v1/twitch/update")
	if err != nil {
		return err
	}

	type Body struct {
		Game  string `json:"game"`
		Title string `json:"title"`
		Login string `json:"login"`
	}

	ts, err := sc.twitchGetSettings()
	if err != nil {
		return err
	}

	if !ts.Update {
		log.Println("twitchUpdateInfo called but twitch updates disabled in settings")
		return nil
	}
	body := Body{
		Game:  game,
		Title: title,
		Login: sc.base.Settings.S.TwitchUpdateChannel,
	}

	result, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", uri.String(), bytes.NewReader(result))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", sc.socialAuth.key)

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 204 {
		errRes := SocialAuthErrorResponse{}
		err = json.NewDecoder(res.Body).Decode(&errRes)
		if err != nil {
			return fmt.Errorf("error decoding error body into SocialAuthErrorResponse")
		}
		return fmt.Errorf("non 204 status code returned from social auth. got: %v (%v) message: %v", errRes.Status, errRes.Error, errRes.Message);
	}

	return nil
}

// TwitchUpdateInfo will update the game and title for the connected twitch account
func (sc Controller) TwitchUpdateInfo(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := sc.twitchUpdateInfo()
	if err != nil {
		sc.base.Response("", "error sending twitch update", http.StatusInternalServerError, w)
		log.Println("Error updating twitch settings: ", err)
		return
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

	// TODO: handle error
	tmpl, err := template.New("run").Parse(ts.TemplateString)
	if err != nil {
		return err.Error()
	}

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
	Update         bool   `json:"update"`
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

func (sc Controller) TwitchPlayCommercial(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	commercialTimes := map[int]bool{30: true, 60: true, 90: true, 120: true, 150: true, 180: true}

	body := struct {
		Length int `json:"length"`
	}{}

	json.NewDecoder(r.Body).Decode(&body)

	defer r.Body.Close()

	if commercialTimes[body.Length] {

		req, err := http.NewRequest("POST", sc.socialAuth.url+"/twitch/commercial?login="+sc.base.Settings.S.TwitchUpdateChannel+"&length="+strconv.Itoa(body.Length), nil)
		if err != nil {
			sc.base.Response("", "error creating run commercial request", 500, w)
			return
		}
		req.Header.Add("Authorization", sc.socialAuth.key)
		res, err := sc.base.HTTPClient.Do(req)
		if err != nil {
			sc.base.Response("", "can't send run commercial request", 500, w)
			return

		}

		if res.StatusCode != 200 {
			sc.base.Response("", "non 200 status code returned while trying to start commercial", 500, w)
			return
		}

		sc.base.Response("ok", "", http.StatusOK, w)
	} else {
		sc.base.Response("", "invalid commerical time", http.StatusBadRequest, w)
	}

}

// TwitchCheckForAuth will check if there is an access token available. It doesn't necessairly say if it's expired or invalid
func (sc Controller) TwitchCheckForAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	avail, err := sc.checkSocialAuth()
	if err != nil {
		sc.base.LogError("while checking for social auth avail", err, true)
		return
	}

	if avail.Twitch {
		sc.base.Response("true", "", 200, w)
	} else {
		sc.base.Response(sc.socialAuth.url, "", 200, w)
	}
}

// TwitchDeleteToken will delete and revoke the twitch token
func (sc Controller) TwitchDeleteToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: implement this once functionality available in social_auth
	w.WriteHeader(http.StatusNoContent)
}
