package social

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-redis/redis"

	"github.com/dghubble/oauth1"

	"github.com/julienschmidt/httprouter"
)

func (sc Controller) TwitterOAuthURL(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	requestToken, _, _ := sc.twitterInfo.RequestToken()
	authorizationURL, _ := sc.twitterInfo.AuthorizationURL(requestToken)

	sc.base.Response(authorizationURL.String(), "", 200, w)
}

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

func (sc Controller) TwitterCheckForAuth(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, err := sc.base.RedisClient.Get("twitterAuth").Result()
	if err == redis.Nil {
		sc.base.Response("false", "", 200, w)
		return
	}

	sc.base.Response("true", "", 200, w)
}

func (sc Controller) TwitterDeleteToken(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	sc.base.RedisClient.Del("twitterAuth")
	w.WriteHeader(http.StatusNoContent)
}