package social

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/go-redis/redis"

	"github.com/dghubble/oauth1"

	"github.com/julienschmidt/httprouter"
)

// TwitterOAuthURL will give the twitter oauth url
func (sc Controller) TwitterOAuthURL(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestToken, _, _ := sc.twitterInfo.RequestToken()
	authorizationURL, _ := sc.twitterInfo.AuthorizationURL(requestToken)

	sc.base.Response(authorizationURL.String(), "", 200, w)
}

// TwitterCallback generates a new twitter accessToken and accessSecret
func (sc Controller) TwitterCallback(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestToken, verifier, _ := oauth1.ParseAuthorizationCallback(r)

	accessToken, accessSecret, _ := sc.twitterInfo.AccessToken(requestToken, "", verifier)

	token := oauth1.NewToken(accessToken, accessSecret)

	b, _ := json.Marshal(token)

	err := sc.base.RedisClient.Set("twitterAuth", b, 0).Err()
	if err != nil {
		log.Printf("Error in setting auth info, err: %v", err)
		sc.base.Response("", "error", 500, w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TwitterCheckForAuth will check if the user is authenticated. It actually just checks if the twitterAuth exists it can be wrong tho idk
func (sc Controller) TwitterCheckForAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, err := sc.base.RedisClient.Get("twitterAuth").Result()
	if err == redis.Nil {
		sc.base.Response("false", "", 200, w)
		return
	}

	sc.base.Response("true", "", 200, w)
}

// TwitterDeleteToken will delete a twitter token from redis
func (sc Controller) TwitterDeleteToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	sc.base.RedisClient.Del("twitterAuth")
	w.WriteHeader(http.StatusNoContent)
}

// TwitterSendUpdate will send the update tweet at the start of a new run
func (sc Controller) TwitterSendUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	err := sc.twitterSendUpdate()
	if err != nil {
		sc.base.Response("", "error sending tweet", http.StatusInternalServerError, w)
	}
}

func (sc Controller) twitterSendUpdate() error {
	// res, err := sc.base.RedisClient.Get("twitterAuth").Bytes()
	// if err != nil {
	// 	sc.base.LogError("Error getting twitter auth from redis", err, true)
	// }

	// t := oauth1.Token{}

	// json.Unmarshal(res, &t)

	// c := sc.twitterInfo.Client(oauth1.NoContext, &t)
	// httpRes, err := c.Post("https://api.twitter.com/1.1/statuses/update.json?status=test", "", nil)
	// if err != nil {
	// 	sc.base.LogError("Error sending tweet", err, true)
	// }

	// fmt.Fprint(w, httpRes.Status)

	t, err := sc.twitterExecuteTemplate()
	if err != nil {
		return err
	}

	fmt.Println(t)
	return nil
}

type TwitterSettings struct {
	SendTweets bool `json:"sendTweets"`
}

func (sc Controller) TwitterSetSettings(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	body := TwitterSettings{}

	json.NewDecoder(r.Body).Decode(&body)

	err := sc.base.RedisClient.Set("twitterSettings", strconv.FormatBool(body.SendTweets), 0).Err()
	if err != nil {
		sc.base.Response("", "error saving settings", http.StatusInternalServerError, w)
		return
	}
}

func (sc Controller) TwitterGetSettings(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

}
