package social

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-redis/redis"

	"github.com/julienschmidt/httprouter"
)

// TwitterCheckForAuth will query socialAuth service is twitch authentication data exists. A true here doesn't necessairly mean that the data is valid but only that it exists
func (sc Controller) TwitterCheckForAuth(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	avail, err := sc.checkSocialAuth()
	if err != nil {
		sc.base.LogError("while checking for social auth avail", err, true)
		return
	}

	if avail.Twitter {
		sc.base.Response("true", "", 200, w)
	} else {
		sc.base.Response(sc.socialAuth.url, "", 200, w)
	}
}

// TwitterDeleteToken will tell socialAuth to delete the twitter auth data
func (sc Controller) TwitterDeleteToken(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// TODO: when implemented at social auth make this work
	w.WriteHeader(http.StatusNoContent)
}

// TwitterSendUpdate will send the update tweet at the start of a new run
func (sc Controller) TwitterSendUpdate(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	err := sc.twitterSendUpdate()
	if err != nil {
		sc.base.Response("", "error sending tweet", http.StatusInternalServerError, w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sc Controller) twitterSendUpdate() error {
	ts, err := sc.twitterExecuteTemplate()
	if err != nil {
		return err
	}

	tweetBody := struct {
		Body string `json:"body,omitempty"`
	}{
		Body: ts,
	}

	url := sc.socialAuth.url + "/api/v1/tweet"
	b, err := json.Marshal(&tweetBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(b))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", sc.socialAuth.key)

	res, err := sc.base.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	if res.StatusCode != 204 {
		return fmt.Errorf("non 200 status code returned from twitter. Status code is %v", res.StatusCode)
	}

	return nil
}

// TwitterSettings contains the settings for Twitter
type TwitterSettings struct {
	SendTweets bool `json:"sendTweets"`
}

// TwitterSetSettings is used to set settings for twitter
func (sc Controller) TwitterSetSettings(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := TwitterSettings{}

	json.NewDecoder(r.Body).Decode(&body)

	err := sc.base.RedisClient.Set("twitterSettings", strconv.FormatBool(body.SendTweets), 0).Err()
	if err != nil {
		sc.base.Response("", "error saving settings", http.StatusInternalServerError, w)
		return
	}
}

// TwitterGetSettings is used to get settings for twitter
func (sc Controller) TwitterGetSettings(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	res, err := sc.base.RedisClient.Get("twitterSettings").Bytes()
	if err != nil {
		if err == redis.Nil {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		sc.base.Response("", "error getting settings", http.StatusInternalServerError, w)
	}
	b, _ := strconv.ParseBool(string(res))
	s := struct {
		SendUpdates bool `json:"sendUpdates"`
	}{b}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s)
}
