package social

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"net/http"
	"strconv"
	"text/template"

	"gopkg.in/mgo.v2/bson"

	"github.com/go-redis/redis"
	"github.com/onestay/MarathonTools-API/api/models"

	"github.com/julienschmidt/httprouter"
)

type twitterTemplate struct {
	Text        string        `json:"text,omitempty"`
	ForMultiple bool          `json:"forMultiple,omitempty"`
	ForRun      bson.ObjectId `json:"forRun,omitempty"`
}

type twitterTemplateOptions struct {
	Game     string
	Runner   []models.PlayerInfo
	Platform string
	Estimate string
	Category string
}

type twitterTemplates []twitterTemplate

// TwitterAddTemplate will add a template to redis
func (sc Controller) TwitterAddTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	t := twitterTemplate{}
	json.NewDecoder(r.Body).Decode(&t)
	defer r.Body.Close()

	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		ts := make([]twitterTemplate, 1)
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

	ta := twitterTemplates{}

	json.Unmarshal(b, &ta)

	ta = append(ta, t)

	mRes, _ := json.Marshal(ta)

	sc.base.RedisClient.Set("twitterTemplates", mRes, 0).Err()
	if err != nil {
		sc.base.Response("", "Error adding template", http.StatusInternalServerError, w)
	}

	sc.TwitterGetTemplates(w, r, ps)
}

// TwitterGetTemplates will return all templates from redis
func (sc Controller) TwitterGetTemplates(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	t, err := sc.twitterGetTemplates()
	if err != nil {
		if err.Error() == "No templates added" {
			sc.base.Response("", err.Error(), http.StatusOK, w)
			return
		}
		sc.base.Response("", err.Error(), http.StatusInternalServerError, w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(t)
}

func (sc Controller) twitterGetTemplates() (*twitterTemplates, error) {
	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		return nil, errors.New("no templates added")
	} else if err != nil {
		return nil, errors.New("error getting templates from redis")
	}

	t := twitterTemplates{}

	json.Unmarshal(b, &t)

	return &t, nil
}

// TwitterDeleteTemplate will delete a template given by the index
func (sc Controller) TwitterDeleteTemplate(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	b, err := sc.base.RedisClient.Get("twitterTemplates").Bytes()
	if err == redis.Nil {
		sc.base.Response("", "No templates added", http.StatusNotFound, w)
		return
	} else if err != nil {
		sc.base.Response("", "Error getting templates from redis", http.StatusInternalServerError, w)
		return
	}

	t := twitterTemplates{}

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

func (sc Controller) twitterExecuteTemplate() (string, error) {

	c := sc.base.CurrentRun
	t := twitterTemplateOptions{c.GameInfo.GameName, c.Players, c.RunInfo.Platform, c.RunInfo.Estimate, c.RunInfo.Category}
	templates, err := sc.twitterGetTemplates()
	if err != nil {
		return "", err
	}
	if len(*templates) == 0 {
		return "", errors.New("no templates added")
	}
	rTemplate := (*templates)[rand.Intn(len(*templates))]

	templ, err := template.New("tweet").Parse(rTemplate.Text)
	if err != nil {
		return "", err
	}

	var execTemplate bytes.Buffer
	err = templ.Execute(&execTemplate, t)
	if err != nil {
		return "", err
	}

	return execTemplate.String(), nil
}
