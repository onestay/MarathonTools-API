package common

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-redis/redis"

	"github.com/julienschmidt/httprouter"
)

// Checklist provides the implamentation of a checklist
type Checklist struct {
	Items map[string]bool
	b     *Controller
}

// NewChecklist initizalies and returns a new Checklist
func NewChecklist(b *Controller) *Checklist {
	var cl map[string]bool
	// a checklist can be initalized in three ways
	// 1: through a checklist file
	// 2: through a checklist saved in redis
	// 3: a new checklist

	if _, err := os.Stat("./config/checklist.json"); err == nil {
		log.Println("Found checklist file. Importing from file.")
		clFile, err := os.Open("./config/checklist.json")
		if err != nil {
			b.LogError("Couldn't open checklist file", err, false)
			return &Checklist{
				Items: make(map[string]bool, 100),
				b:     b,
			}
		}

		defer clFile.Close()

		json.NewDecoder(clFile).Decode(&cl)

		err = os.Rename("./config/checklist.json", "./config/checklist_imported.json")
		if err != nil {
			b.LogError("when renaming. Please rename manually", err, false)
		}
	} else if b, err := b.RedisClient.Get("checklist").Bytes(); err != redis.Nil && len(b) != 0 {
		// if a checklist was already saved on a previous run we want to keep that
		// however it is likely that we want it set to completely false in that case
		log.Println("Checklist found in redis. Loading checklist from redis")
		json.Unmarshal(b, &cl)
		for k := range cl {
			cl[k] = false
		}
	} else {
		log.Println("Not checklist file found. Checklist not found in Redis. Creating new checklist")
		cl = make(map[string]bool, 100)
	}

	c := Checklist{
		Items: cl,
		b:     b,
	}

	c.saveToRedis()

	return &c
}

// AddItem will add an item to the checklist
func (c *Checklist) AddItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		c.Items[i] = false
		c.saveToRedis()
		json.NewEncoder(w).Encode(c.Items)
		go c.b.WSChecklistUpdate()
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// DeleteItem will delete an item from the checklist
func (c *Checklist) DeleteItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		delete(c.Items, i)
		c.saveToRedis()
		json.NewEncoder(w).Encode(c.Items)
		go c.b.WSChecklistUpdate()
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// ToggleItem will toggle the status of an item if it exists
func (c *Checklist) ToggleItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		r, e := c.Items[i]
		if e {
			c.Items[i] = !r
		}

		go c.b.WSChecklistUpdate()
		json.NewEncoder(w).Encode(c.Items)
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// CheckDoneHTTP will return whether all the items in the checklist are done
func (c *Checklist) CheckDoneHTTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	c.b.Response(strconv.FormatBool(c.CheckDone()), "", http.StatusOK, w)
}

// CheckDone will return whether all the items in the checklist are done
func (c *Checklist) CheckDone() bool {
	for _, v := range c.Items {
		if !v {
			return false
		}
	}
	return true
}

//GetChecklist will get all items from the checlist
func (c *Checklist) GetChecklist(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	json.NewEncoder(w).Encode(c.Items)
}

func (c *Checklist) saveToRedis() {
	b, _ := json.Marshal(c.Items)
	err := c.b.RedisClient.Set("checklist", b, 0).Err()
	if err != nil {
		c.b.LogError("while saving checklist to redis", err, false)
	}
}