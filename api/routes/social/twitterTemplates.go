package social

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gopkg.in/mgo.v2/bson"

	"github.com/go-redis/redis"

	"github.com/julienschmidt/httprouter"
)

type TwitterTemplate struct {
	Text        string        `json:"text,omitempty"`
	ForMultiple bool          `json:"forMultiple,omitempty"`
	ForRun      bson.ObjectId `json:"forRun,omitempty"`
}

type TwitterTemplates []TwitterTemplate

func (sc Controller) TwitterAddTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	t := TwitterTemplate{}
	json.NewDecoder(r.Body).Decode(&t)
	defer r.Body.Close()

	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		ts := make([]TwitterTemplate, 1)
		ts[0] = t

		mRes, _ := json.Marshal(ts)

		err := sc.base.RedisClient.Set("twitterTemplates", mRes, 0).Err()
		if err != nil {
			sc.base.Response("", "Error adding template", http.StatusInternalServerError, w)
			return
		}
		sc.TwitterGetTemplates(w, r, ps)
		return
	} else if err != nil {
		sc.base.Response("", "error getting templates from redis", http.StatusInternalServerError, w)
		return
	}

	ta := TwitterTemplates{}

	json.Unmarshal(b, &ta)

	ta = append(ta, t)

	mRes, _ := json.Marshal(ta)

	sc.base.RedisClient.Set("twitterTemplates", mRes, 0).Err()
	if err != nil {
		sc.base.Response("", "Error adding template", http.StatusInternalServerError, w)
	}

	sc.TwitterGetTemplates(w, r, ps)
}

func (sc Controller) TwitterGetTemplates(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		sc.base.Response("", "No templates added", http.StatusNotFound, w)
		return
	} else if err != nil {
		sc.base.Response("", "Error getting templates from redis", http.StatusInternalServerError, w)
		return
	}

	t := TwitterTemplates{}

	json.Unmarshal(b, &t)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (sc Controller) TwitterDeleteTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		sc.base.Response("", "No templates added", http.StatusNotFound, w)
		return
	} else if err != nil {
		sc.base.Response("", "Error getting templates from redis", http.StatusInternalServerError, w)
		return
	}

	t := TwitterTemplates{}

	json.Unmarshal(b, &t)

	i, err := strconv.Atoi(ps.ByName("index"))
	if err != nil {
		sc.base.Response("", "index isn't a valid int", http.StatusBadRequest, w)
	}

	t = append(t[:i], t[i+1:]...)

	mRes, _ := json.Marshal(t)

	sc.base.RedisClient.Set("twitterTemplates", mRes, 0).Err()
	if err != nil {
		sc.base.Response("", "Error adding template", http.StatusInternalServerError, w)
	}

	sc.TwitterGetTemplates(w, r, ps)
}
