package common

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	"github.com/julienschmidt/httprouter"
)

type item struct {
	Key  string `json:"key"`
	Done bool   `Json:"done"`
}

// Checklist provides the implamentation of a checklist
type Checklist struct {
	Items []*item
	// we need to access this at some crucial times like timer start. we can't afford the time it takes for the loop to finish processing then
	// that's why we set this variable with every call to add, remove and toggle so this variable will only have to be accessed to see if the checklist is done
	Finished bool
	b        *Controller
}

// NewChecklist initizalies and returns a new Checklist
func NewChecklist(b *Controller) *Checklist {

	var items []*item
	// // a checklist can be initalized in three ways
	// // 1: through a checklist file
	// // 2: through a checklist saved in redis
	// // 3: a new checklist

	if _, err := os.Stat("./config/checklist.json"); err == nil {
		log.Println("Found checklist file. Importing from file.")
		clFile, err := os.Open("./config/checklist.json")
		if err != nil {
			b.LogError("Couldn't open checklist file", err, false)
			items = make([]*item, 0)
			return &Checklist{
				Items: items,
				b:     b,
			}
		}

		defer clFile.Close()

		json.NewDecoder(clFile).Decode(&items)

		err = os.Rename("./config/checklist.json", "./config/checklist_imported.json")
		if err != nil {
			b.LogError("when renaming. Please rename manually", err, false)
		}
	} else if b, err := b.RedisClient.Get("checklist").Bytes(); err != redis.Nil && len(b) != 0 {
		// if a checklist was already saved on a previous run we want to keep that
		// however it is likely that we want it set to completely false in that case
		log.Println("Checklist found in redis. Loading checklist from redis")
		json.Unmarshal(b, &items)
		for _, v := range items {
			v.Done = false
		}
	} else {
		log.Println("Not checklist file found. Checklist not found in Redis. Creating new checklist")
		items = make([]*item, 0)
	}

	c := Checklist{
		Items: items,
		b:     b,
	}
	c.Finished = false
	c.saveToRedis()

	return &c
}

// AddItem will add an item to the checklist
func (c *Checklist) AddItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		if c.getItemIndex(i) != -1 {
			c.b.Response("", "Item already exists", http.StatusBadRequest, w)
			return
		}
		itemObj := item{
			Key:  i,
			Done: false,
		}

		c.Items = append(c.Items, &itemObj)
		go func() {
			c.Finished = c.CheckDone()
			c.saveToRedis()
		}()
		json.NewEncoder(w).Encode(c.Items)
		go c.b.WSChecklistUpdate()
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// DeleteItem will delete an item from the checklist
func (c *Checklist) DeleteItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		index := c.getItemIndex(i)

		if index == -1 {
			c.b.Response("", "Item doesn't exist", http.StatusBadRequest, w)
			return
		}
		copy(c.Items[index:], c.Items[index+1:])
		c.Items[len(c.Items)-1] = nil
		c.Items = c.Items[:len(c.Items)-1]

		go func() {
			c.Finished = c.CheckDone()
			c.saveToRedis()
		}()
		json.NewEncoder(w).Encode(c.Items)
		go c.b.WSChecklistUpdate()
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// ToggleItem will toggle the status of an item if it exists
func (c *Checklist) ToggleItem(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if i := r.URL.Query().Get("item"); len(i) != 0 {
		item := c.getItem(i)

		if item == nil {
			c.b.Response("", "Item doesn't exist", http.StatusBadRequest, w)
			return
		}

		item.Done = !item.Done

		go c.b.WSChecklistUpdate()
		json.NewEncoder(w).Encode(c.Items)
		c.Finished = c.CheckDone()
		return
	}
	c.b.Response("", "no item defined", http.StatusBadRequest, w)
}

// CheckDoneHTTP will return whether all the items in the checklist are done
func (c *Checklist) CheckDoneHTTP(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	res := struct {
		Ok   bool `json:"ok"`
		Data bool `json:"data"`
	}{true, c.CheckDone()}

	json.NewEncoder(w).Encode(res)
}

// CheckDone will return whether all the items in the checklist are done
func (c *Checklist) CheckDone() bool {
	for _, v := range c.Items {
		if !v.Done {
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

func (c *Checklist) getItem(key string) *item {
	for _, item := range c.Items {
		if item.Key == key {
			return item
		}
	}

	return nil
}

func (c *Checklist) getItemIndex(key string) int {
	for i, item := range c.Items {
		if item.Key == key {
			return i
		}
	}

	return -1
}
