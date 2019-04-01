package social

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/go-redis/redis"
	"github.com/julienschmidt/httprouter"
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
	go func() {
		ts := TwitchSettings{}

		ser, _ := json.Marshal(ts)

		sc.base.RedisClient.Set("twitchSettings", ser, 0)
	}()
	b, _ := json.Marshal(resStruct)

	err = sc.base.RedisClient.Set("twitchAuth", b, 0).Err()
	if err != nil {
		log.Printf("Error in setting auth info, err: %v", err)
		sc.base.Response("", "error", 500, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (sc Controller) twitchUpdateChannelID() {
	tr := TwitchResponse{}

	b, err := sc.base.RedisClient.Get("twitchAuth").Bytes()
	if err != nil {
		if err == redis.Nil {
			log.Println("tried updating twitch update channel but no twitch auth data")
			return
		}
		log.Println("tried updating twitch update channel, error while getting auth data", err)
		return
	}

	json.Unmarshal(b, &tr)

	resChan := make(chan bool)

	go sc.getChannelID(resChan, &tr)
	<-resChan

	bU, _ := json.Marshal(tr)

	err = sc.base.RedisClient.Set("twitchAuth", bU, 0).Err()
	if err != nil {
		log.Printf("Error in setting auth info after updatechannelid, err: %v", err)
		return
	}

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

func (sc Controller) twitchRefreshToken() (string, error) {
	b, err := sc.base.RedisClient.Get("twitchAuth").Bytes()
	if err != nil {
		return "", err
	}
	t := TwitchResponse{}

	json.Unmarshal(b, &t)

	var uri *url.URL
	uri, _ = url.Parse(refreshTokenURL)

	parameters := url.Values{}
	parameters.Add("client_id", sc.twitchInfo.ClientID)
	parameters.Add("client_secret", sc.twitchInfo.ClientSecret)
	parameters.Add("grant_type", "refresh_token")
	parameters.Add("refresh_token", t.RefreshToken)
	uri.RawQuery = parameters.Encode()

	req, err := http.NewRequest("POST", uri.String(), nil)
	if err != nil {
		return "", err
	}

	res, err := sc.base.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", fmt.Errorf("Expected code 200 but got %v from twitch while trying to get refresh token", res.StatusCode)
	}

	refreshResponse := struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Scope        string `json:"scope"`
	}{}
	json.NewDecoder(res.Body).Decode(&refreshResponse)

	t.AccessToken = refreshResponse.AccessToken
	t.Scope = refreshResponse.Scope
	t.RefreshToken = refreshResponse.RefreshToken
	bMar, _ := json.Marshal(t)

	err = sc.base.RedisClient.Set("twitchAuth", bMar, 0).Err()
	if err != nil {
		return "", err
	}

	return t.AccessToken, nil
}
